package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client

func ConnectMongoDB(ctx context.Context, uri string) error {
	var err error
	clientOpts := options.Client().ApplyURI(uri)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	Client, err = mongo.Connect(ctx, clientOpts)
	if err != nil {
		return err
	}

	err = Client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	log.Println("Connected to MongoDB")
	return nil
}
