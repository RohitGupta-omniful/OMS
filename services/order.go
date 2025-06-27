package services

import (
	"context"

	"github.com/RohitGupta-omniful/OMS/db"
	"github.com/RohitGupta-omniful/OMS/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrderService struct{}

type OrderServiceInterface interface {
	UpdateOrderStatus(ctx context.Context, orderID string, newStatus string) error
	UpsertOrder(ctx context.Context, order models.Order) error
}

// NewOrderService creates and returns a new OrderService instance.
func NewOrderService() *OrderService {
	return &OrderService{}
}

// UpdateOrderStatus updates the status of an order based on orderID.
func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID string, newStatus string) error {
	_, err := db.OrderCollection().UpdateOne(
		ctx,
		bson.M{"order_id": orderID},
		bson.M{"$set": bson.M{"status": newStatus}},
	)
	return err
}

// UpsertOrder inserts or updates an order in the database.
func (s *OrderService) UpsertOrder(ctx context.Context, order models.Order) error {
	_, err := db.OrderCollection().UpdateOne(
		ctx,
		bson.M{"order_id": order.OrderID},
		bson.M{"$set": order},
		options.Update().SetUpsert(true),
	)
	return err
}
