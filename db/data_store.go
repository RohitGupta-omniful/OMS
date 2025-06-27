package db

import "go.mongodb.org/mongo-driver/mongo"

func WebhookCollection() *mongo.Collection {
	return Client.Database("oms").Collection("webhooks")
}

func OrderCollection() *mongo.Collection {
	return Client.Database("oms").Collection("orders")
}
