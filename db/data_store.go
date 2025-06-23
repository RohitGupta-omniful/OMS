package db

import "go.mongodb.org/mongo-driver/mongo"

func OrderCollection() *mongo.Collection {
	return Client.Database("oms").Collection("orders")
}
