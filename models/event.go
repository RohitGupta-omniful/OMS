package models

type OrderCreatedEvent struct {
	OrderID string  `json:"order_id"`
	SKUID   string  `json:"sku_id"`
	HubID   string  `json:"hub_id"`
	Qty     int     `json:"quantity"`
	Price   float64 `json:"price"`
	Status  string  `json:"status,omitempty"`
}
