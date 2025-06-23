package models

type Order struct {
	OrderID      string  `bson:"order_id"`
	CustomerName string  `bson:"customer_name"`
	HubID        string  `bson:"hub_id"`
	SKUID        string  `bson:"sku_id"`
	Qty          int     `bson:"qty"`
	Price        float64 `bson:"price"`
	Status       string  `bson:"status"`
}
