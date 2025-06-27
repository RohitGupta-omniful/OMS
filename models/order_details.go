package models

type Order struct {
	OrderID      string  `json:"order_id" bson:"order_id"`
	CustomerName string  `json:"customer_name" bson:"customer_name"`
	HubID        string  `json:"hub_id" bson:"hub_id"`
	SKUID        string  `json:"sku_id" bson:"sku_id"`
	Qty          int     `json:"qty" bson:"qty"`
	Price        float64 `json:"price" bson:"price"`
	Status       string  `json:"status" bson:"status"`
	CustomerID   int     `json:"customer_id" bson:"customer_id"`
}
