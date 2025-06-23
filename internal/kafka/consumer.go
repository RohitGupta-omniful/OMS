package kafka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/RohitGupta-omniful/OMS/models"
	"github.com/google/uuid"
	"github.com/omniful/go_commons/kafka"
	"github.com/omniful/go_commons/pubsub"
	"github.com/omniful/go_commons/pubsub/interceptor"
)

type InventoryUpdateRequest struct {
	SKUID           string `json:"sku_id"`
	HubID           string `json:"hub_id"`
	QuantityChange  int    `json:"quantity_change"`
	TransactionType string `json:"transaction_type"`
}

type OrderConsumer struct{}

// Process implements the pubsub.IPubSubMessageHandler interface.
func (oc *OrderConsumer) Process(ctx context.Context, msg *pubsub.Message) error {
	log.Printf("Received Kafka message - Topic: %s, Key: %s", msg.Topic, string(msg.Key))

	// ⬇️ Updated to match producer's struct
	var evt models.OrderCreatedEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		log.Printf("Failed to unmarshal Kafka message: %v", err)
		return err
	}

	log.Printf("Order Event - SKUID: %s, HubID: %s, Qty: %d", evt.SKUID, evt.HubID, evt.Qty)

	// Validate UUID format
	if _, err := uuid.Parse(evt.SKUID); err != nil {
		log.Printf("Invalid SKUID in Kafka event: %v", err)
		return err
	}
	if _, err := uuid.Parse(evt.HubID); err != nil {
		log.Printf("Invalid HubID in Kafka event: %v", err)
		return err
	}

	update := InventoryUpdateRequest{
		SKUID:           evt.SKUID,
		HubID:           evt.HubID,
		QuantityChange:  evt.Qty,  // positive quantity
		TransactionType: "remove", // indicate removal
	}

	body, err := json.Marshal(update)
	if err != nil {
		log.Printf("Failed to marshal inventory update request: %v", err)
		return err
	}

	req, err := http.NewRequest("POST", "http://localhost:8000/inventory/update", bytes.NewReader(body))
	if err != nil {
		log.Printf("Failed to create HTTP request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 5 * time.Second}

	// Retry HTTP request up to 3 times
	const maxRetries = 3
	backoff := time.Second

	for i := 0; i < maxRetries; i++ {
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("HTTP request attempt %d failed: %v", i+1, err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Println("Inventory update success")
				return nil
			}
			respBody, _ := io.ReadAll(resp.Body)
			log.Printf("Inventory service returned status %d, response: %s", resp.StatusCode, string(respBody))
		}

		if i < maxRetries-1 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	return fmt.Errorf("failed to update inventory after %d attempts", maxRetries)
}

func InitConsumer(ctx context.Context, topic string) {
	consumer := kafka.NewConsumer(
		kafka.WithBrokers([]string{"localhost:9092"}),
		kafka.WithConsumerGroup("oms-service"),
		kafka.WithClientID("oms-consumer"),
		kafka.WithKafkaVersion("2.8.1"),
	)

	consumer.SetInterceptor(interceptor.NewRelicInterceptor())
	consumer.RegisterHandler(topic, &OrderConsumer{})

	log.Printf("Kafka consumer subscribed to topic: %s", topic)
	go consumer.Subscribe(ctx)

	select {}
}
