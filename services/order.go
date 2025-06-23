package services

import (
	"context"

	"github.com/RohitGupta-omniful/OMS/db"
	"github.com/RohitGupta-omniful/OMS/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrderService struct{}

// Constructor without KafkaProducer
func NewOrderService() *OrderService {
	return &OrderService{}
}

func (s *OrderService) UpsertOrder(ctx context.Context, order models.Order) error {
	_, err := db.OrderCollection().UpdateOne(
		ctx,
		bson.M{"order_id": order.OrderID},
		bson.M{"$setOnInsert": order},
		options.Update().SetUpsert(true),
	)

	return err
}
