
# OMS - Order Management System Microservice

This microservice handles **bulk order uploads**, **validation**, **persistence**, and **event emission** (via **Kafka** and **SQS**) for the **Omniful Order Management Platform**.

---

## Project Structure

```
OMS/
├── configs/          # Configuration loaders and environment bindings
├── db/               # Database connection and migrations
├── IMS_APIS/         # Third-party/internal IMS service integrations
├── internal/         # Core logic (CSV parsing, validation, SQS handling)
├── kafka/            # Kafka producer setup
├── localstack/       # LocalStack setup for local AWS service mocks
├── middleware/       # Custom Gin middleware (auth, logging, i18n)
├── models/           # Structs for DB, Kafka messages, and API contracts
├── public/           # Static files (e.g., invalid order CSVs)
├── route/            # API route definitions
├── server/           # HTTP server startup
├── services/         # Business logic (OrderService, etc.)
├── webhook/          # Webhook handlers (if applicable)
├── docker-compose.yml # Docker-based local dev environment
├── go.mod / go.sum   # Go module and dependency management
├── main.go           # Application entry point
└── sample.csv        # Example order CSV file
```

---

## Features

- Upload bulk orders via **S3-hosted CSV**
- Validate CSV structure and enforce business rules:
  - `hub_id` and `sku_id` checks via **IMS**
- Insert valid orders into **MongoDB**
- Save invalid rows into a downloadable CSV file
- Send order events to **Kafka** (`order.created`)
- Push jobs to **AWS SQS** (`CreateBulkOrderQueue`)
- Local development with **Docker** + **LocalStack**
- Token-based **authentication middleware**
- i18n-ready (log messages and responses can be localized)

---

## Authentication

All APIs require an authorization header:

```
Authorization: Bearer secret-token
```

---

## 📡 API Reference

### `POST /api/orders/upload`

Uploads the S3 file path for processing.

#### Required Headers

| Key            | Value               |
|----------------|---------------------|
| Authorization  | Bearer secret-token |
| Content-Type   | application/json    |

#### Request Body

```json
{
  "s3_path": "s3://oms-temp-public/sample.csv"
}
```

#### Example Response

```json
{
  "message": "CSV validated and SQS event published successfully",
  "published_to": "CreateBulkOrderQueue",
  "payload": "{\"bucket\":\"oms-temp-public\",\"key\":\"sample.csv\"}"
}
```

---

##  Middleware

### Auth Middleware

Validates the `Authorization` header.

```go
Authorization: Bearer secret-token
```

Rejects the request with a `400 Bad Request` if invalid or missing.

---

## i18n Support

Log messages and responses use the i18n key-based translation model, which supports future multi-language translation and standardized messaging throughout the microservice.

---

## Local Development

Use Docker and LocalStack to spin up local services:

```bash
docker-compose up -d
go run main.go
```

This will simulate SQS, S3, Kafka, and MongoDB locally.

---

## Testing with Postman / Thunder Client

### Headers

```http
Authorization: Bearer secret-token
Content-Type: application/json
```

### Body

```json
{
  "s3_path": "s3://oms-temp-public/sample.csv"
}
```

### Response

```json
{
  "message": "CSV validated and SQS event published successfully"
}
```

---

## Kafka Topic

- **Topic**: `order.created`
- Publishes an event when an order passes validation and is ready for fulfillment.

---

## Example: Invalid Orders

Invalid rows are stored in a generated CSV at:

```
/public/invalid_orders_<timestamp>.csv
```

You can download the file to review failed entries.
