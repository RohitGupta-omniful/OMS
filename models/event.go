package models

type OrderCreatedEvent struct {
	OrderID    string  `json:"order_id" bson:"order_id"`
	SKUID      string  `json:"sku_id" bson:"sku_id"`
	HubID      string  `json:"hub_id" bson:"hub_id"`
	Qty        int     `json:"quantity" bson:"quantity"`
	Price      float64 `json:"price" bson:"price"`
	Status     string  `json:"status,omitempty" bson:"status,omitempty"`
	CustomerID int     `json:"customer_id" bson:"customer_id"`
}
