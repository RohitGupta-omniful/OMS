package kafka

import (
	"context"
	"encoding/json"

	"github.com/omniful/go_commons/kafka"
	"github.com/omniful/go_commons/pubsub"
	"go.uber.org/zap"
)

type Producer struct {
	client pubsub.Publisher
	topic  string
	logger *zap.SugaredLogger
}

func NewProducerWithConfig(topic string, brokers []string) *Producer {
	client := kafka.NewProducer(
		kafka.WithBrokers(brokers),
		kafka.WithClientID("oms-producer"),
		kafka.WithKafkaVersion("2.8.1"),
	)

	return &Producer{
		client: client,
		topic:  topic,
		logger: zap.S().With("component", "KafkaProducer"),
	}
}

// Emit sends a message to the configured Kafka topic.
func (p *Producer) Emit(ctx context.Context, key string, event any) error {
	value, err := json.Marshal(event)
	if err != nil {
		p.logger.Errorf("failed to marshal event: %v", err)
		return err
	}
	p.logger.Infof("Publishing to topic=%s, key=%s, value=%s", p.topic, key, string(value))

	msg := &pubsub.Message{
		Topic:   p.topic,
		Key:     key,
		Value:   value,
		Headers: map[string]string{"source": "oms"},
	}

	p.logger.Infof("Publishing to topic=%s, key=%s", p.topic, key)
	return p.client.Publish(ctx, msg)
}

func (p *Producer) Close() {
	if p.client != nil {
		p.client.Close()
	}
}
