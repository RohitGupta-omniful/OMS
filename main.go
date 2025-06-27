package main

import (
	"time"

	"github.com/RohitGupta-omniful/OMS/IMS_APIS"
	"github.com/RohitGupta-omniful/OMS/db"
	"github.com/RohitGupta-omniful/OMS/internal/handlers"
	"github.com/RohitGupta-omniful/OMS/kafka"
	"github.com/RohitGupta-omniful/OMS/server"
	"github.com/RohitGupta-omniful/OMS/services"
	"github.com/RohitGupta-omniful/OMS/webkooks"
	"github.com/omniful/go_commons/config"
	"github.com/omniful/go_commons/i18n"
	"github.com/omniful/go_commons/log"
	"github.com/omniful/go_commons/s3"
)

func main() {
	// Initialize config
	if err := config.Init(15 * time.Second); err != nil {
		log.Errorf("Failed to initialize config: %v", err)
		return
	}

	// Get context with config
	ctx, err := config.TODOContext()
	if err != nil {
		log.Errorf(i18n.Translate(ctx, "failed_config_context %v"), err)
		return
	}

	// Initialize IMS client
	if err := IMS_APIS.InitIMSClient(ctx); err != nil {
		log.Errorf(i18n.Translate(ctx, "failed_connect_ims %v"), err)
		return
	}

	// Connect to MongoDB
	mongoURI := config.GetString(ctx, "mongodb.uri")
	if err := db.ConnectMongoDB(ctx, mongoURI); err != nil {
		log.Errorf(i18n.Translate(ctx, "failed_connect_mongo %v"), err)
		return
	}

	// Initialize webhook collection
	webkooks.SetWebhookCollection(db.WebhookCollection())

	// Initialize S3 client
	s3Client, err := s3.NewDefaultAWSS3Client()
	if err != nil {
		log.Errorf(i18n.Translate(ctx, "failed_init_s3 %v"), err)
		return
	}

	// Create handler with S3 client
	handler := handlers.NewHandler(ctx, s3Client)

	// Initialize HTTP server
	app := server.Initialize(ctx, handler)

	// Order service
	orderService := services.NewOrderService()

	// Start CSV Processor
	go handlers.StartCSVProcessor(ctx, *s3Client, orderService)

	// Start Kafka consumer with orderService injected
	go kafka.InitConsumer(ctx, "order.created", orderService)

	// Start HTTP server
	serverName := config.GetString(ctx, "server.name")
	serverPort := config.GetString(ctx, "server.port")
	log.Infof(i18n.Translate(ctx, "starting_server_on_port"), serverName, serverPort)

	if err := app.StartServer(serverPort); err != nil {
		log.Errorf(i18n.Translate(ctx, "server_failed %v"), err)
	}
}
