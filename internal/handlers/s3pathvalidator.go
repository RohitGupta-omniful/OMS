package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/RohitGupta-omniful/OMS/IMS_APIS"
	"github.com/RohitGupta-omniful/OMS/internal/SQS"
	"github.com/RohitGupta-omniful/OMS/internal/kafka"
	"github.com/RohitGupta-omniful/OMS/models"
	"github.com/RohitGupta-omniful/OMS/services"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/omniful/go_commons/config"
	"github.com/omniful/go_commons/csv"
	"github.com/omniful/go_commons/log"
	"github.com/omniful/go_commons/sqs"
)

type UploadRequest struct {
	S3Path string `json:"s3_path"`
}

type BulkOrderRequest struct {
	Path   string `json:"path"`
	Bucket string `json:"bucket"`
}

func (h *Handler) UploadCSV(c *gin.Context) {
	var req UploadRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if !strings.HasPrefix(req.S3Path, "s3://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid s3_path format"})
		return
	}

	bucket, key := parseS3Path(req.S3Path)

	_, err := h.S3Client.HeadObject(c.Request.Context(), &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		fmt.Printf("[S3 Error] %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error accessing S3"})
		return
	}

	publisher, err := SQS.PublishCreateBulkOrderEvent(c.Request.Context())
	if err != nil {
		fmt.Printf("[SQS Error] Failed to publish event: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to push event to queue"})
		return
	}

	payload := fmt.Sprintf(`{"bucket":"%s", "key":"%s"}`, bucket, key)
	msg := &sqs.Message{
		Value: []byte(payload),
	}

	err = publisher.Publish(c.Request.Context(), msg)
	if err != nil {
		fmt.Printf("[SQS Error] Failed to publish message: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish message to queue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "CSV validated and SQS event published successfully",
		"published_to": "CreateBulkOrderQueue",
		"payload":      payload,
	})
}

func parseS3Path(s3Path string) (bucket string, key string) {
	trimmed := strings.TrimPrefix(s3Path, "s3://")
	parts := strings.SplitN(trimmed, "/", 2)
	return parts[0], parts[1]
}

func StartCSVProcessor(ctx context.Context, s3Client s3.Client, orderService *services.OrderService) {
	logger := log.DefaultLogger()
	kafkaProducer := kafka.NewProducerWithConfig("order.created", []string{"localhost:9092"}) // ← 💥 Error Here

	queueURL := config.GetString(ctx, "sqs.bulkOrderQueueUrl")
	queueName := path.Base(queueURL)
	logger.Infof("Listening to SQS queue: %s", queueName)

	sqsCfg := &sqs.Config{
		Account:  config.GetString(ctx, "sqs.account"),
		Endpoint: config.GetString(ctx, "sqs.endpoint"),
		Region:   config.GetString(ctx, "aws.region"),
	}

	qObj, err := sqs.NewStandardQueue(ctx, queueName, sqsCfg)
	if err != nil {
		logger.Panicf("Failed to create SQS queue: %v", err)
	}

	consumer, err := sqs.NewConsumer(
		qObj,
		uint64(config.GetInt(ctx, "sqs.consumer.workerCount")),
		uint64(1),
		&queueHandler{
			S3Client:      s3Client,
			OrderService:  orderService,
			SQSQueue:      qObj,
			KafkaProducer: kafkaProducer,
		},
		int64(config.GetInt(ctx, "sqs.consumer.batchSize")),
		int64(config.GetDuration(ctx, "sqs.consumer.visibilityTimeout").Seconds()),
		false,
		false,
	)

	if err != nil {
		logger.Panicf("Failed to start SQS consumer: %v", err)
	}

	consumer.Start(ctx)
}

type queueHandler struct {
	S3Client      s3.Client
	OrderService  *services.OrderService
	SQSQueue      *sqs.Queue
	KafkaProducer *kafka.Producer
}

func (h *queueHandler) Process(ctx context.Context, msgs *[]sqs.Message) error {
	logger := log.DefaultLogger()

	for _, msg := range *msgs {
		var evt struct {
			Bucket string `json:"bucket"`
			Key    string `json:"key"`
		}

		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			logger.Errorf("Invalid SQS JSON: %v", err)
			continue
		}

		logger.Infof("Processing file: s3://%s/%s", evt.Bucket, evt.Key)

		getObjOutput, err := h.S3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &evt.Bucket,
			Key:    &evt.Key,
		})
		if err != nil {
			logger.Errorf("Failed to download CSV from S3: %v", err)
			continue
		}
		defer getObjOutput.Body.Close()

		tmpFile := filepath.Join(os.TempDir(), filepath.Base(evt.Key))
		outFile, err := os.Create(tmpFile)
		if err != nil {
			logger.Errorf("Failed to create temp file: %v", err)
			continue
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, getObjOutput.Body)
		if err != nil {
			logger.Errorf("Failed to write S3 object to file: %v", err)
			continue
		}

		csvReader, err := csv.NewCommonCSV(
			csv.WithBatchSize(100),
			csv.WithSource(csv.Local),
			csv.WithLocalFileInfo(tmpFile),
			csv.WithHeaderSanitizers(csv.SanitizeAsterisks, csv.SanitizeToLower),
			csv.WithDataRowSanitizers(csv.SanitizeSpace, csv.SanitizeToLower),
		)
		if err != nil {
			logger.Errorf("Failed to create CSV reader: %v", err)
			continue
		}

		if err := csvReader.InitializeReader(ctx); err != nil {
			logger.Errorf("Failed to initialize CSV reader: %v", err)
			continue
		}

		headers, err := csvReader.GetHeaders()
		if err != nil {
			logger.Errorf("Failed to read CSV headers: %v", err)
			continue
		}
		logger.Infof("CSV Headers: %v", headers)

		colIdx := make(map[string]int)
		for i, col := range headers {
			colIdx[col] = i
		}

		var invalid csv.Records

		for !csvReader.IsEOF() {
			records, err := csvReader.ReadNextBatch()
			if err != nil {
				logger.Errorf("Failed to read CSV batch: %v", err)
				break
			}

			for _, row := range records {
				logger.Infof("Processing CSV row: %v", row)

				hubID := strings.TrimSpace(row[colIdx["hub_id"]])
				skuID := strings.TrimSpace(row[colIdx["sku_id"]])
				qtyStr := strings.TrimSpace(row[colIdx["quantity"]])
				priceStr := strings.TrimSpace(row[colIdx["price"]])
				orderID := strings.TrimSpace(row[colIdx["order_id"]])
				customerName := strings.TrimSpace(row[colIdx["customer_name"]])

				qty, err := strconv.Atoi(qtyStr)
				price, perr := strconv.ParseFloat(priceStr, 64)

				// Validate fields
				if err != nil || qty <= 0 || perr != nil || price < 0 {
					logger.Warnf("Invalid quantity or price in row: %v", row)
					invalid = append(invalid, row)
					continue
				}

				if len(skuID) == 0 || len(hubID) == 0 {
					logger.Warnf("Empty skuID or hubID in row: %v", row)
					invalid = append(invalid, row)
					continue
				}

				if !IMS_APIS.ValidateHub(hubID) {
					logger.Warnf("Invalid hubID in row: %v", row)
					invalid = append(invalid, row)
					continue
				}
				if !IMS_APIS.ValidateSKUOnHub(skuID) {
					logger.Warnf("Invalid skuID on hub in row: %v", row)
					invalid = append(invalid, row)
					continue
				}

				order := models.Order{
					OrderID:      orderID,
					CustomerName: customerName,
					HubID:        hubID,
					SKUID:        skuID,
					Qty:          qty,
					Price:        price,
					Status:       "on_hold",
				}

				if err := h.OrderService.UpsertOrder(ctx, order); err != nil {
					logger.Errorf("Failed to upsert order: %v", err)
					invalid = append(invalid, row)
					continue
				}

				event := models.OrderCreatedEvent{
					OrderID: order.OrderID,
					SKUID:   order.SKUID,
					HubID:   order.HubID,
					Qty:     order.Qty,
					Price:   order.Price,
				}

				// Additional validation before emitting event
				if event.SKUID == "" || event.HubID == "" || event.Qty <= 0 {
					logger.Warnf("Skipping Kafka event due to invalid event fields: %+v", event)
					continue
				}

				logger.Infof("Emitting Kafka event: %+v", event)
				if err := h.KafkaProducer.Emit(ctx, "order.created", event); err != nil {
					logger.Errorf("Failed to emit Kafka event for order_id %s: %v", order.OrderID, err)
				} else {
					logger.Infof("Kafka event emitted for order_id: %s", order.OrderID)
				}
			}
		}

		if len(invalid) > 0 {
			logger.Warnf("Found %d invalid rows in CSV", len(invalid))
		}
	}

	return nil
}
