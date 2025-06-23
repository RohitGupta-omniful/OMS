package main

import (
	"log"
	"time"

	"github.com/RohitGupta-omniful/OMS/IMS_APIS"
	"github.com/RohitGupta-omniful/OMS/db"
	"github.com/RohitGupta-omniful/OMS/internal/handlers"
	"github.com/RohitGupta-omniful/OMS/internal/kafka"
	"github.com/RohitGupta-omniful/OMS/server"
	"github.com/RohitGupta-omniful/OMS/services"
	"github.com/omniful/go_commons/config"
	"github.com/omniful/go_commons/s3"
)

func main() {
	// Initialize config
	err := config.Init(15 * time.Second)
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// Get context with config
	ctx, err := config.TODOContext()
	if err != nil {
		log.Fatalf("Failed to load config context: %v", err)
	}
	//fmt.Println("DEBUG kafka.brokers =", config.GetStringSlice(ctx, "kafka.brokers"))

	// Initialize IMS client
	err = IMS_APIS.InitIMSClient(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to IMS: %v", err)
	}

	// Connect to MongoDB
	mongoURI := config.GetString(ctx, "mongodb.uri")
	err = db.ConnectMongoDB(ctx, mongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Initialize S3 client
	s3Client, err := s3.NewDefaultAWSS3Client()
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}

	// Create handler with S3 client
	handler := handlers.NewHandler(s3Client)

	// Initialize HTTP server
	app := server.Initialize(ctx, handler)

	// Initialize OrderService (Kafka producer will be created inside StartCSVProcessor)
	orderService := services.NewOrderService()

	// Start CSV Processor (handles SQS consumption and Kafka production)
	go handlers.StartCSVProcessor(ctx, *s3Client, orderService)

	go kafka.InitConsumer(ctx, "order.created")

	// Start HTTP server
	serverName := config.GetString(ctx, "server.name")
	serverPort := config.GetString(ctx, "server.port")
	log.Printf("Starting %s on %s", serverName, serverPort)

	if err := app.StartServer(serverPort); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
