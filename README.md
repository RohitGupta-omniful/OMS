#  OMS - Order Management System Microservice

This microservice handles bulk order uploads, validation, persistence, and event emission (via Kafka and SQS) for the Omniful Order Management platform.

---

## Project Structure

├── configs/ # Configuration loaders and env bindings
├── db/ # DB connection and migrations
├── IMS_APIS/ # Third-party/internal service integrations
├── internal/ # Core logic (e.g., CSV parsing, validation and SQS)
├── kafka/ # Kafka producer setup
├── localstack/ # Localstack test support (SQS, S3)
├── middleware/ # Custom Gin middleware (auth, logging)
├── models/ # Structs used in DB, Kafka, and API
├── public/ # Static files like invalid order CSVs
├── route/ # API route definitions
├── server/ # HTTP server startup logic
├── services/ # Business logic (e.g., OrderService)
├── webhook/ # Webhook handlers
├── docker-compose.yml # Dev containers for Kafka/SQS/DB/etc
├── go.mod / go.sum # Go module and dependencies
├── main.go # App entry point
└── sample.csv # Example CSV file


## Features

- Upload bulk orders from S3-hosted CSV
- Validate CSV structure and business rules (e.g. hub and sku validation)
- Insert valid orders to DB
- Save invalid rows to downloadable CSV
- Send order events to Kafka (`order.created`)
- Push jobs to SQS (`CreateBulkOrderQueue`)
- Local setup with Docker & Localstack
- Middleware support for token-based auth and logging
- i18n support (localizable logs/responses)

---


## API Reference

- POST /api/orders/upload

Uploads a CSV path from S3 for processing.

##Headers:
   Key : Value
  Authorization: Bearer secret-token
  Content-Type: application/json

## Body:
- json
{
  "s3_path": "s3://oms-temp-public/sample.csv"
}


Response:
json
{
  "message": "CSV validated and SQS event published successfully",
  "published_to": "CreateBulkOrderQueue",
  "payload": "{\"bucket\":\"oms-temp-public\",\"key\":\"sample.csv\"}"
}

Auth Middleware
   Authorization: Bearer secret-token

   
