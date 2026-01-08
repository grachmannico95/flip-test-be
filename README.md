# FLIP TEST BACKEND

Go HTTP service for asynchronous CSV statement processing with event-driven architecture, worker pool, and clean architecture patterns.

## Architecture Overview

### Components

- Domain Layer (`internal/domain`)
  - Core business models (Transaction, Upload)
  - Repository interface (storage abstraction)
  - Domain errors
- Storage Layer (`internal/storage`)
  - In-memory store with `sync.RWMutex`
  - Thread-safe operations
  - Idempotency tracking
- Event Bus (`internal/eventbus`)
  - Channel-based event bus
  - Worker pool with configurable size
  - Retry with exponential backoff
  - Idempotent event processing
- Service Layer (`internal/service`)
  - CSV streaming processor
  - Statement service (business logic)
  - Event publishing
- Handler Layer (`internal/handler`)
  - HTTP request/response handling
  - Input validation
  - Error mapping
- Middleware (`internal/middleware`)
  - Request ID injection (trace_id)
  - Structured logging

### Data Flow

```
HTTP -> Handler -> Service -> Goroutine -> CSV Processor -> Event Bus -> Worker Pool -> Repository
```

## Trades-off

- In-Memory Storage vs Database:
  - Pros:
    - Simple, no external dependencies
    - Fast read/write operations
    - Perfect for prototype/demo
    - Easy to swap with DB via Repository interface
  - Cons:
    - Data lost on restart
    - Not horizontally scalable
    - Limited by memory size
- Channel-Based Event Bus vs Message Queue
  - Pros:
    - Simple, no external broker (Kafka/RabbitMQ)
    - Fast, low latency
    - Suitable for single instance
  - Cons:
    - Events lost on crash
    - Not distributed
    - Bounded by channel buffer size
- CSV Streaming vs Batch Processing
  - Pros:
    - Constant memory usage
    - Handles large files (GB+)
    - Immediate event publishing
    - No file size limit
  - Cons:
    - Slightly more complex than batch
- Async Upload vs Sync Processing
  - Pros:
    - Better UX (immediate response)
    - Server doesn't block
    - Handles slow uploads
  - Cons:
    - Client needs to poll for completion
- Idempotency via Event ID
  - Pros:
    - Prevents duplicate processing
    - Simple to implement
    - Effective for retry scenarios
  - Cons:
    - Memory grows with events
    - Lost on restart
- Exponential Backoff Retry
  - Pros:
    - Handles transient failures
    - Reduces load during outages
    - Standard industry practice
  - Cons:
    - May delay processing
    - Need to tune max retries

## How to run

1. Clone the repository
   ```
   git clone https://github.com/grachmannico95/flip-test-be
   ```
2. Install dependencies
   ```
   go mod download
   ```
3. Copy environment file
   ```
   cp .env.example .env
   ```
4. Run the server
   ```
   go run cmd/server/main.go
   ```
   or
   ```
   make run
   ```

## How to run unit and integration test

```
make test
```

or

```
make test-race
```

## Example CSV file
[sample.csv](./test/file/sample.csv)

## API Documentation
[postman collection](./postman/flip-test-be.postman_collection.json)

### cURL

- POST /statements
  ```
  curl --location 'http://localhost:8080/statements' \
  --form 'file=@"/Users/gustirachmannico/Project/go/flip-test-be/test/file/sample.csv"'
  ```
  response:
  ```
  {
      "status": "processing",
      "upload_id": "a2a90ca1-548a-49b2-bd49-5eee399a6140"
  }
  ```
- GET /balance?upload_id=
  ```
  curl "http://localhost:8080/balance?upload_id=a2a90ca1-548a-49b2-bd49-5eee399a6140"
  ```
  response:
  ```
  {
      "balance": 2150000,
      "upload_id": "a2a90ca1-548a-49b2-bd49-5eee399a6140"
  }
  ```
- GET /transactions/issues?upload_id=
  ```
  curl --location 'http://localhost:8080/transactions/issues?upload_id=a2a90ca1-548a-49b2-bd49-5eee399a6140&page=1&per_page=10'
  ```
  response:
  ```
  {
      "items": [
          {
              "timestamp": 1674507885,
              "counterparty": "BOB SMITH",
              "type": "DEBIT",
              "amount": 100000,
              "status": "FAILED",
              "description": "invalid transaction",
              "line_number": 3
          },
          {
              "timestamp": 1674507886,
              "counterparty": "ALICE WONDER",
              "type": "CREDIT",
              "amount": 300000,
              "status": "PENDING",
              "description": "pending payment",
              "line_number": 4
          },
          {
              "timestamp": 1674507889,
              "counterparty": "EVE WILSON",
              "type": "DEBIT",
              "amount": 150000,
              "status": "FAILED",
              "description": "insufficient funds",
              "line_number": 7
          },
          {
              "timestamp": 1674507891,
              "counterparty": "GRACE LEE",
              "type": "DEBIT",
              "amount": 80000,
              "status": "PENDING",
              "description": "awaiting approval",
              "line_number": 9
          }
      ],
      "page": 1,
      "per_page": 10,
      "total": 4,
      "upload_id": "a2a90ca1-548a-49b2-bd49-5eee399a6140"
  }
  ```

## Log example

```
{"level":"info","timestamp":"2026-01-08T10:06:43.546+0700","caller":"logger/logger.go:108","msg":"Handling upload request","trace_id":"8be50637-9bc2-4c19-9dd4-d589c76ddcac"}
{"level":"info","timestamp":"2026-01-08T10:06:43.546+0700","caller":"logger/logger.go:108","msg":"Creating upload record","trace_id":"8be50637-9bc2-4c19-9dd4-d589c76ddcac","upload_id":"c69b3e09-f5dc-4875-b5b2-9fb856ab0594"}
{"level":"info","timestamp":"2026-01-08T10:06:43.546+0700","caller":"logger/logger.go:108","msg":"Upload created, processing started","trace_id":"8be50637-9bc2-4c19-9dd4-d589c76ddcac","upload_id":"c69b3e09-f5dc-4875-b5b2-9fb856ab0594"}
{"level":"info","timestamp":"2026-01-08T10:06:43.546+0700","caller":"logger/logger.go:108","msg":"Upload successful","trace_id":"8be50637-9bc2-4c19-9dd4-d589c76ddcac","upload_id":"c69b3e09-f5dc-4875-b5b2-9fb856ab0594"}
{"level":"info","timestamp":"2026-01-08T10:06:43.546+0700","caller":"logger/logger.go:108","msg":"HTTP request","trace_id":"8be50637-9bc2-4c19-9dd4-d589c76ddcac","method":"POST","path":"/statements","status":202,"duration_ms":0,"remote_addr":"[::1]:52442"}
{"level":"info","timestamp":"2026-01-08T10:06:43.546+0700","caller":"logger/logger.go:108","msg":"Starting async CSV processing","upload_id":"c69b3e09-f5dc-4875-b5b2-9fb856ab0594"}
{"level":"info","timestamp":"2026-01-08T10:06:43.547+0700","caller":"logger/logger.go:108","msg":"Starting CSV processing","upload_id":"c69b3e09-f5dc-4875-b5b2-9fb856ab0594"}
{"level":"info","timestamp":"2026-01-08T10:06:43.547+0700","caller":"logger/logger.go:108","msg":"CSV processing completed","upload_id":"c69b3e09-f5dc-4875-b5b2-9fb856ab0594","total_lines":10,"success_count":10,"error_count":0}
{"level":"info","timestamp":"2026-01-08T10:06:43.547+0700","caller":"logger/logger.go:108","msg":"CSV processing completed successfully","upload_id":"c69b3e09-f5dc-4875-b5b2-9fb856ab0594"}
```
