package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/RohitGupta-omniful/OMS/IMS_APIS"
	"github.com/RohitGupta-omniful/OMS/internal/SQS"
	"github.com/RohitGupta-omniful/OMS/kafka"
	"github.com/RohitGupta-omniful/OMS/models"
	"github.com/RohitGupta-omniful/OMS/services"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/omniful/go_commons/config"
	"github.com/omniful/go_commons/csv"
	"github.com/omniful/go_commons/i18n"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.Translate(c, "invalid request body")})
		return
	}

	if !strings.HasPrefix(req.S3Path, "s3://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.Translate(c, "invalid s3_path format")})
		return
	}

	bucket, key := parseS3Path(req.S3Path)

	_, err := h.S3Client.HeadObject(c.Request.Context(), &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		log.Errorf(i18n.Translate(c, "error accessing S3: %v"), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.Translate(c, "error accessing S3")})
		return
	}

	publisher, err := SQS.PublishCreateBulkOrderEvent(c.Request.Context())
	if err != nil {
		log.Errorf(i18n.Translate(c, "failed to publish event to SQS: %v"), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.Translate(c, "failed to push event to queue")})
		return
	}

	payload := `{"bucket":"` + bucket + `", "key":"` + key + `"}`

	msg := &sqs.Message{
		Value: []byte(payload),
	}

	if err = publisher.Publish(c.Request.Context(), msg); err != nil {
		log.Errorf(i18n.Translate(c, "failed to publish message to queue: %v"), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.Translate(c, "failed to publish message to queue")})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      i18n.Translate(c, "CSV validated and SQS event published successfully"),
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
	kafkaProducer := kafka.NewProducerWithConfig("order.created", []string{"localhost:9092"})

	queueURL := config.GetString(ctx, "sqs.bulkOrderQueueUrl")
	queueName := path.Base(queueURL)
	logger.Infof(i18n.Translate(ctx, "Listening to SQS queue: %s"), queueName)

	sqsCfg := &sqs.Config{
		Account:  config.GetString(ctx, "sqs.account"),
		Endpoint: config.GetString(ctx, "sqs.endpoint"),
		Region:   config.GetString(ctx, "aws.region"),
	}

	qObj, err := sqs.NewStandardQueue(ctx, queueName, sqsCfg)
	if err != nil {
		logger.Panicf(i18n.Translate(ctx, "failed to create SQS queue: %v"), err)
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
		logger.Panicf(i18n.Translate(ctx, "failed to start SQS consumer: %v"), err)
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
			logger.Errorf(i18n.Translate(ctx, "invalid SQS JSON: %v"), err)
			continue
		}

		logger.Infof(i18n.Translate(ctx, "processing file: s3://%s/%s"), evt.Bucket, evt.Key)

		getObjOutput, err := h.S3Client.GetObject(ctx, &s3.GetObjectInput{Bucket: &evt.Bucket, Key: &evt.Key})
		if err != nil {
			logger.Errorf(i18n.Translate(ctx, "failed to download CSV from S3: %v"), err)
			continue
		}
		defer getObjOutput.Body.Close()

		tmpFile := filepath.Join(os.TempDir(), filepath.Base(evt.Key))
		outFile, err := os.Create(tmpFile)
		if err != nil {
			logger.Errorf(i18n.Translate(ctx, "failed to create temp file: %v"), err)
			continue
		}
		defer outFile.Close()

		if _, err = io.Copy(outFile, getObjOutput.Body); err != nil {
			logger.Errorf(i18n.Translate(ctx, "failed to write S3 object to file: %v"), err)
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
			logger.Errorf(i18n.Translate(ctx, "failed to create CSV reader: %v"), err)
			continue
		}
		if err = csvReader.InitializeReader(ctx); err != nil {
			logger.Errorf(i18n.Translate(ctx, "failed to initialize CSV reader: %v"), err)
			continue
		}

		headers, err := csvReader.GetHeaders()
		if err != nil {
			logger.Errorf(i18n.Translate(ctx, "failed to read CSV headers: %v"), err)
			continue
		}
		logger.Infof(i18n.Translate(ctx, "CSV headers: %v"), headers)

		colIdx := make(map[string]int)
		for i, col := range headers {
			colIdx[col] = i
		}

		var invalid csv.Records

		for !csvReader.IsEOF() {
			records, err := csvReader.ReadNextBatch()
			if err != nil {
				logger.Errorf(i18n.Translate(ctx, "failed to read CSV batch: %v"), err)
				break
			}

			for _, row := range records {
				logger.Infof(i18n.Translate(ctx, "processing CSV row: %v"), row)

				hubID := strings.TrimSpace(row[colIdx["hub_id"]])
				skuID := strings.TrimSpace(row[colIdx["sku_id"]])
				qtyStr := strings.TrimSpace(row[colIdx["quantity"]])
				priceStr := strings.TrimSpace(row[colIdx["price"]])
				orderID := strings.TrimSpace(row[colIdx["order_id"]])
				customerName := strings.TrimSpace(row[colIdx["customer_name"]])
				customerIDStr := strings.TrimSpace(row[colIdx["tenant_id"]])

				qty, err := strconv.Atoi(qtyStr)
				price, perr := strconv.ParseFloat(priceStr, 64)
				customerID, cerr := strconv.Atoi(customerIDStr)

				if err != nil || qty <= 0 || perr != nil || price < 0 || cerr != nil || customerID <= 0 {
					logger.Warnf(i18n.Translate(ctx, "invalid data in row: %v"), row)
					invalid = append(invalid, row)
					continue
				}

				if hubID == "" || skuID == "" {
					logger.Warnf(i18n.Translate(ctx, "empty skuID or hubID in row: %v"), row)
					invalid = append(invalid, row)
					continue
				}

				if !IMS_APIS.ValidateHub(ctx, hubID) {
					logger.Warnf(i18n.Translate(ctx, "invalid hubID in row: %v"), row)
					invalid = append(invalid, row)
					continue
				}

				if !IMS_APIS.ValidateSKUOnHub(ctx, skuID) {
					logger.Warnf(i18n.Translate(ctx, "invalid skuID on hub in row: %v"), row)
					invalid = append(invalid, row)
					continue
				}

				order := models.Order{OrderID: orderID, CustomerName: customerName, HubID: hubID, SKUID: skuID, Qty: qty, Price: price, Status: "on_hold", CustomerID: customerID}

				if err := h.OrderService.UpsertOrder(ctx, order); err != nil {
					logger.Errorf(i18n.Translate(ctx, "failed to upsert order: %v"), err)
					invalid = append(invalid, row)
					continue
				}

				event := models.OrderCreatedEvent{OrderID: order.OrderID, SKUID: order.SKUID, HubID: order.HubID, Qty: order.Qty, Price: order.Price, CustomerID: order.CustomerID}

				if event.SKUID == "" || event.HubID == "" || event.Qty <= 0 {
					logger.Warnf(i18n.Translate(ctx, "skipping Kafka event due to invalid event fields: %+v"), event)
					continue
				}

				logger.Infof(i18n.Translate(ctx, "emitting Kafka event: %+v"), event)
				if err := h.KafkaProducer.Emit(ctx, "order.created", event); err != nil {
					logger.Errorf(i18n.Translate(ctx, "failed to emit Kafka event for order_id %s: %v"), event.OrderID, err)
				} else {
					logger.Infof(i18n.Translate(ctx, "Kafka event emitted for order_id: %s"), event.OrderID)
				}
			}
		}

		if len(invalid) > 0 {
			timestamp := time.Now().Format("20060102_150405")
			filePath := "public/invalid_orders_" + timestamp + ".csv"

			dest := &csv.Destination{}
			dest.SetFileName(filePath)
			dest.SetUploadDirectory("public/")
			dest.SetRandomizedFileName(false)

			writer, err := csv.NewCommonCSVWriter(
				csv.WithWriterHeaders(headers),
				csv.WithWriterDestination(*dest),
			)
			if err != nil {
				logger.Errorf(i18n.Translate(ctx, "failed to create CSV writer: %v"), err)
				return err
			}
			defer writer.Close(ctx)

			if err := writer.Initialize(); err != nil {
				logger.Errorf(i18n.Translate(ctx, "failed to initialize CSV writer: %v"), err)
				return err
			}

			if err := writer.WriteNextBatch(invalid); err != nil {
				logger.Errorf(i18n.Translate(ctx, "failed to write invalid rows: %v"), err)
				return err
			}

			logger.Infof(i18n.Translate(ctx, "invalid rows saved to CSV at: %s"), filePath)
			publicURL := "http://localhost:8082/" + filePath
			logger.Infof(i18n.Translate(ctx, "download invalid CSV here: %s"), publicURL)
		}
	}

	return nil
}
