
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
## Project Workflow

This project processes order data from CSV files uploaded to S3 and integrates with SQS, Kafka, MongoDB, and an internal IMS (Inventory Management System) for full order lifecycle management.

---

### 1. **CSV Upload via API**
- User sends a POST request to `POST /api/orders/upload`.
- The request includes a JSON body with the S3 path of the uploaded CSV.
- Middleware handles:
  - Authorization using a Bearer token.
  - Request logging.
  - i18n support for standardized messaging.

---

### 2. **CSV Validation**
- The service fetches the CSV file from the given S3 path.
- Parses and validates:
  - Structure (columns like `hub_id`, `sku_id`, `quantity`).
  - Data types and required fields.
- Valid and invalid rows are separated:
  - Valid rows → sent for further processing.
  - Invalid rows → written to a new CSV and uploaded back to S3.

---

### 3. **SQS Message Publishing**
- If CSV validation succeeds, a message with S3 file metadata is published to the AWS SQS queue: `CreateBulkOrderQueue`.
- Response to user confirms:
  - Queue message published.
  - Path of the file processed.

---

### 4. **SQS Consumer (Worker)**
- A worker service listens to the `CreateBulkOrderQueue`.
- For each received message:
  - Downloads and parses the CSV file from S3.
  - Iterates through each row to:
    - Extract `hub_id`, `sku_id`, `quantity`.
    - Call IMS API to validate `hub_id` and `sku_id`.

---

### 5. **Order Creation**
- If the row is valid:
  - An order is inserted into MongoDB with status `"on_hold"`.
  - An event is published to the Kafka topic: `order.created`.

- If the row is invalid:
  - It’s ignored or added to a separate invalid file for auditing.

---

### 6. **Kafka Consumer (Order Finalization)**
- Another service listens to `order.created` Kafka events.
- For each event:
  - Calls IMS again to check inventory availability.
  - If sufficient:
    - IMS reserves/deducts inventory.
    - Order status in MongoDB is updated to `"new_Order"`.
  - If insufficient:
    - Order remains `"on_hold"` or flagged for manual review.

---

### 7. **Invalid Order Handling**
- Invalid rows are exported to a CSV.
- This CSV is uploaded to the `/public/` directory in S3.
- Can be downloaded later by the user for correction and re-upload.

---

### Summary of Services

| Component        | Purpose |
|------------------|---------|
| **API Layer**    | Accept CSV path, validate headers, initiate workflow. |
| **S3**           | Stores input and output CSV files. |
| **SQS**          | Manages decoupled communication between CSV ingestion and processing services. |
| **Worker**       | Consumes messages, validates rows, interacts with IMS. |
| **IMS APIs**     | Validate hub/sku and check inventory levels. |
| **Kafka**        | Publishes `order.created` events for inventory management. |
| **MongoDB**      | Stores and tracks the status of each order. |

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
