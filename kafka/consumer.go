package kafka

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/RohitGupta-omniful/OMS/models"
	"github.com/RohitGupta-omniful/OMS/services"
	"github.com/RohitGupta-omniful/OMS/webkooks"
	"github.com/google/uuid"
	"github.com/omniful/go_commons/i18n"
	"github.com/omniful/go_commons/kafka"
	"github.com/omniful/go_commons/log"
	"github.com/omniful/go_commons/pubsub"
	"github.com/omniful/go_commons/pubsub/interceptor"
)

type InventoryUpdateRequest struct {
	SKUID           string `json:"sku_id"`
	HubID           string `json:"hub_id"`
	QuantityChange  int    `json:"quantity_change"`
	TransactionType string `json:"transaction_type"`
}

type OrderConsumer struct {
	OrderService services.OrderServiceInterface
}

func (oc *OrderConsumer) Process(ctx context.Context, msg *pubsub.Message) error {
	log.Infof(i18n.Translate(ctx, "Received Kafka message - Topic: %s, Key: %s"), msg.Topic, string(msg.Key))

	var evt models.OrderCreatedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		log.Errorf(i18n.Translate(ctx, "Failed to unmarshal Kafka message: %v"), err)
		return err
	}

	log.Infof(i18n.Translate(ctx, "Order Event - OrderID: %s, SKUID: %s, HubID: %s, Qty: %d"), evt.OrderID, evt.SKUID, evt.HubID, evt.Qty)

	// Validate UUIDs
	if _, err := uuid.Parse(evt.SKUID); err != nil {
		log.Errorf(i18n.Translate(ctx, "Invalid SKUID in Kafka event: %v"), err)
		return err
	}
	if _, err := uuid.Parse(evt.HubID); err != nil {
		log.Errorf(i18n.Translate(ctx, "Invalid HubID in Kafka event: %v"), err)
		return err
	}

	update := InventoryUpdateRequest{
		SKUID:           evt.SKUID,
		HubID:           evt.HubID,
		QuantityChange:  evt.Qty,
		TransactionType: "remove",
	}

	body, err := json.Marshal(update)
	if err != nil {
		log.Errorf(i18n.Translate(ctx, "Failed to marshal inventory update request: %v"), err)
		return err
	}

	req, err := http.NewRequest("POST", "http://localhost:8000/inventory/update", bytes.NewReader(body))
	if err != nil {
		log.Errorf("Failed to create HTTP request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 5 * time.Second}
	const maxRetries = 3
	backoff := time.Second

	for i := 0; i < maxRetries; i++ {
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf(i18n.Translate(ctx, "HTTP request attempt %d failed: %v"), i+1, err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Infof(i18n.Translate(ctx, "Inventory update succeeded"))

				if err := oc.OrderService.UpdateOrderStatus(ctx, evt.OrderID, "new_order"); err != nil {
					log.Errorf(i18n.Translate(ctx, "Failed to update order status: %v"), err)
					return err
				}

				log.Infof(i18n.Translate(ctx, "Order %s status updated to 'new_order'"), evt.OrderID)
				return nil
			}

			webkooks.NotifyTenantWebhook(ctx, int64(evt.CustomerID), evt)

			respBody, _ := io.ReadAll(resp.Body)
			log.Errorf(i18n.Translate(ctx, "Inventory service returned status %d, response: %s"), resp.StatusCode, string(respBody))
		}

		if i < maxRetries-1 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	log.Errorf(i18n.Translate(ctx, "Failed to update inventory after %d attempts"), maxRetries)
	return err
}

func InitConsumer(ctx context.Context, topic string, orderService services.OrderServiceInterface) {
	consumer := kafka.NewConsumer(
		kafka.WithBrokers([]string{"localhost:9092"}),
		kafka.WithConsumerGroup("oms-service"),
		kafka.WithClientID("oms-consumer"),
		kafka.WithKafkaVersion("2.8.1"),
	)

	consumer.SetInterceptor(interceptor.NewRelicInterceptor())
	consumer.RegisterHandler(topic, &OrderConsumer{
		OrderService: orderService,
	})

	log.Infof(i18n.Translate(ctx, "Kafka consumer subscribed to topic: %s"), topic)
	go consumer.Subscribe(ctx)

	select {}
}
