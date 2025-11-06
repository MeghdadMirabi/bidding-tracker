# Auction Bid Tracker

This is a simple **online auction system** implemented in Go. It allows users to concurrently bid on items for sale and provides a REST API for all auction operations.

---

## Features

- Record a user's bid on an item
- Get the current winning bid for an item
- Get all bids for an item
- Get all items a user has bid on

---

## Prerequisites

- Go 1.22 or later
- No external database required (in-memory storage only)

---

## API Endpoints

| Method | Endpoint | Description |
|--------|---------|------------|
| POST   | `/bids` | Record a new bid |
| GET    | `/items/:item_id/bids` | Get all bids for an item |
| GET    | `/items/:item_id/winning` | Get the current winning bid |
| GET    | `/users/:user_id/items` | Get all items the user has bid on |

---

## Example Items

The server pre-populates 3 example items in the in-memory repository:

| ItemID | Title  | Description    | Starting Price |
|--------|--------|----------------|----------------|
| item1  | title1 | description1   | 100            |
| item2  | title2 | description2   | 200            |
| item3  | title3 | description3   | 150            |

---
### Data Structures

The system uses in-memory storage to manage auction data. The main entities are **User**, **Item**, and **Bid**.

```go
type User struct {
    UserID   string `json:"user_id"`
    Username string `json:"username"`
}

type Item struct {
    ItemID        string  `json:"item_id"`
    Title         string  `json:"title"`
    Description   string  `json:"description"`
    StartingPrice float64 `json:"starting_price"`
}

type Bid struct {
    BidID     string    `json:"bid_id"`
    ItemID    string    `json:"item_id"`
    UserID    string    `json:"user_id"`
    Amount    float64   `json:"amount"`
    CreatedAt time.Time `json:"created_at"`
}

```
---
### Concurrency Approach

The system supports multiple users interacting concurrently. Concurrency control is handled primarily in the **repository layer**, while the **service layer** orchestrates business logic and the **handler layer** exposes HTTP endpoints. The Gin framework spawns a separate goroutine for each incoming HTTP request, allowing concurrent access to the system.


#### Repository Layer (`MemoryRepo`)

The repository stores all shared data in memory and uses a **read/write mutex (`sync.RWMutex`)** to prevent race conditions.  

- **Read operations** (`RLock`) – allow multiple concurrent reads:
  - `GetBidsByItem(itemID string)` – returns all bids for a specific item.  
  - `GetWinningBid(itemID string)` – returns the highest bid for a specific item.  
  - `GetItemsByUser(userID string)` – returns all items a user has bid on.  

- **Write operations** (`Lock`) – ensure exclusive access when modifying shared state:
  - `RecordBidForItem(bid model.Bid)` – records a new bid for an item.  
  - `AddItem(item model.Item)` – adds a new item to the repository (used for initialization or tests).  

The mutex guarantees:
- Concurrent reads do not block each other.  
- Writes are safely serialized, preventing data races.  

#### Service Layer (`BiddingService`)

The service layer provides business logic and interacts with the repository. It **does not use additional locks** because the repository already manages concurrency.  

**Methods:**
- `PlaceBid(itemID, userID string, amount float64)`  
  - Validates and creates a new bid, then calls `MemoryRepo.RecordBidForItem`.  
  - Ensures that bids are correctly linked to both the item and the user.  

- `GetBidsForItem(itemID string)`  
  - Calls `MemoryRepo.GetBidsByItem` to fetch all bids for a given item.  
  - Returns the bids from the repository.  

- `GetWinningBid(itemID string)`  
  - Calls `MemoryRepo.GetWinningBid` to determine the highest bid for the item.  
  - Resolves ties using the earliest bid timestamp.  

- `GetItemsByUser(userID string)`  
  - Calls `MemoryRepo.GetItemsByUser` to retrieve all items a user has bid on.  

> The service layer acts as a **logical bridge** between the HTTP handlers and the repository, encapsulating business rules without handling concurrency directly.


#### Handler Layer (`BiddingHandler`)

The handler layer exposes HTTP endpoints to clients via Gin. Gin spawns a **goroutine per HTTP request**, allowing multiple users to interact with the system simultaneously.  

**Methods and HTTP Endpoints:**
- `RecordBidHandler` → POST `/bids`  
  - Parses the incoming bid request.  
  - Calls `BiddingService.PlaceBid` to create a bid.  
  - Sends a structured JSON response or error using `utils.JSONResponse` / `utils.JSONError`.  

- `GetBidsByItemHandler` → GET `/items/:item_id/bids`  
  - Extracts the `item_id` from the URL.  
  - Calls `BiddingService.GetBidsForItem` to fetch all bids.  
  - Returns a JSON array of bids or an empty array if no bids exist.  

- `GetWinningBidHandler` → GET `/items/:item_id/winning`  
  - Extracts the `item_id` from the URL.  
  - Calls `BiddingService.GetWinningBid` to fetch the highest bid.  
  - Returns JSON with the winning bid or 404 if no bids exist.  

- `GetItemsByUserHandler` → GET `/users/:user_id/items`  
  - Extracts the `user_id` from the URL.  
  - Calls `BiddingService.GetItemsByUser` to fetch all items the user has bid on.  
  - Returns a JSON array of items or an empty array if the user has no bids.  

**Key Points:**
- Each HTTP request runs in a separate goroutine, so multiple clients can interact concurrently.  
- The handlers **do not manage concurrency directly**; they rely on the repository’s mutex.  
- Handlers focus on **request parsing, response formatting, and error handling**.  


#### Summary

| Layer              | Methods / Functions                       | Concurrency Approach                                |
|-------------------|------------------------------------------|----------------------------------------------------|
| **Repository**     | RecordBidForItem, AddItem, GetBidsByItem, GetWinningBid, GetItemsByUser | `Lock` for writes, `RLock` for reads (thread-safe) |
| **Service**        | PlaceBid, GetBidsForItem, GetWinningBid, GetItemsByUser | Delegates to repository; no locks needed           |
| **Handler (Gin)**  | RecordBidHandler, GetBidsByItemHandler, GetWinningBidHandler, GetItemsByUserHandler | Each request runs in its own goroutine; relies on repository for concurrency |

This design ensures **safe concurrent reads and writes**, separates concerns between layers, and allows **highly concurrent HTTP access**.

---
## Unit Tests

The project includes comprehensive unit tests for the repository, service, and handler layers, ensuring correctness, concurrency safety, and proper error handling.

### Repository Layer Tests

The repository layer (`MemoryRepo`) tests cover:

- **Recording Bids**: Ensures valid bids are recorded, handles edge cases such as zero, negative, extremely large amounts, and future/past timestamps.
- **Getting Bids by Item**: Verifies retrieval of all bids for a given item, including items with no bids, non-existing items, and large datasets.
- **Getting Winning Bid**: Determines the highest bid for an item, including tie scenarios, extreme values, and concurrent access.
- **Getting Items by User**: Retrieves all items a user has placed bids on, handling duplicates, large bid volumes, and concurrent access.

The repository tests use **table-driven testing**, **parallel subtests**, and **concurrency tests** with `sync.WaitGroup` to simulate multiple users bidding concurrently.

### Service Layer Tests

The BiddingService tests cover:

- **PlaceBid**: Validates bid placement logic, checking minimum/maximum amounts, empty fields, low bids, and repository errors.
- **GetBidsForItem**: Retrieves all bids for an item, including handling no bids, repository errors, and invalid requests.
- **GetWinningBid**: Confirms correct winning bid is returned, handling errors and edge cases.
- **GetItemsByUser**: Ensures correct items are retrieved for a user, including no items and repository errors.

The service tests use **gomock** for mocking the repository and **table-driven test cases** for all scenarios.

### Handler Layer Tests

The BiddingHandler tests cover API endpoints implemented with Gin:

- **RecordBidHandler**: Tests valid/invalid bid requests, JSON parsing errors, missing fields, invalid amounts, service errors, and concurrency scenarios.
- **GetBidsByItemHandler**: Tests retrieval of bids for a specific item, including valid responses, no bids, invalid item IDs, and service errors.
- **GetWinningBidHandler**: Ensures correct winning bid response and proper error handling.
- **GetItemsByUserHandler**: Validates items retrieval for a user and error handling.
  
Handler tests use **httptest** to simulate HTTP requests and responses, and **parallel subtests** for concurrency scenarios.

---

### Integration Tests

The project includes integration tests to verify the end-to-end behavior of the system. These tests simulate HTTP requests to the API endpoints using an in-memory repository, ensuring that the system works correctly without depending on an external database.

#### Test Approach

1. **Router Setup**  
   Each test initializes a new `gin.Engine` router using an in-memory repository (`MemoryRepo`) and the `BiddingService`. Helper functions are provided to simplify router setup:
   - `SetupTestRouter()` – Initializes the router with an empty repository.
   - `SetupTestRouterWithItems(items ...Item)` – Initializes the router and seeds it with provided items.

2. **Request Execution**  
   Requests are executed and responses parsed using helper functions:
   - `ExecuteRequest()` – Executes an HTTP request and returns the raw response recorder.
   - `ExecuteRequestAndParse()` – Executes an HTTP request and parses the JSON response into a Go map for assertions.

3. **Testing Scenarios**  
   The integration tests cover the main API endpoints:
   - `RecordBidHandler` – Tests placing a bid, including valid bids and invalid JSON input.
   - `GetBidsByItemHandler` – Tests retrieving all bids for a specific item.
   - `GetWinningBidHandler` – Tests retrieving the highest bid for an item, including scenarios where there are no bids or the item does not exist.
   - `GetItemsByUserHandler` – Tests retrieving all items a specific user has bid on, including users with no bids or nonexistent users.

4. **Assertions**  
   The tests use `require` from `testify` to verify:
   - HTTP status codes are as expected.
   - Response payloads contain the correct data (e.g., bid amounts, user IDs, timestamps).
   - Timestamps are valid RFC3339 format.
   - Correct handling of empty results or nonexistent items/users.

5. **Isolation**  
   Each test uses a fresh in-memory repository, ensuring no cross-test interference and full isolation.

This approach ensures that the API behaves correctly under realistic conditions while keeping tests fast and deterministic.

---
### Performance Tests

The project includes performance benchmarks to evaluate the system under different workloads. These tests measure throughput, latency, and memory usage for various bidding and query scenarios, using an in-memory repository.

#### Benchmark Approach

1. **Repository and Service Setup**  
   All performance tests use `MemoryRepo` with `BiddingService`. Each benchmark initializes items and users according to the scenario to simulate realistic load.

2. **Benchmark Types**

   - **PlaceBid - Isolated Items (Low Contention)**  
     Measures bid placement on independent items with no concurrency. This micro-benchmark simulates users bidding on different items simultaneously, ensuring minimal contention.

   - **PlaceBid - Shared Item (High Contention)**  
     Simulates many users placing bids concurrently on a single item to test thread-safety and contention handling. Uses `b.RunParallel()` with atomic operations to ensure consistent bid increments.

   - **GetWinningBid - Single Threaded (Low Contention)**  
     Measures performance of retrieving the winning bid for multiple items sequentially, simulating low read concurrency.

   - **GetWinningBid - Concurrent (High Contention)**  
     Simulates multiple threads concurrently reading the winning bid for the same item, testing the system under high read contention.

   - **Mixed Workload (Concurrent Readers and Writers)**  
     Simulates a realistic scenario with both bid placements (writers) and winning bid queries (readers) on the same item. The workload ratio can be configured (e.g., 70% reads, 30% writes).

   - **Load Scenarios**  
     Configurable scenarios allow testing:
       - Low vs. high contention
       - Read-heavy vs. write-heavy workloads
       - Mixed read/write workloads
       - Burst traffic vs. steady traffic

3. **Metrics Collected**

   - **Throughput** – Total operations per second.
   - **Latency** – Minimum, maximum, average, p95, and p99 latencies for operations.
   - **Memory Usage** – Memory allocated during benchmark.
   - **Success and Failure Counts** – Number of successful bids, failed bids, and read operations per scenario.
   - **Item-Level Stats** – Number of successful bids per item.

4. **Implementation Details**

   - Benchmarks use Go’s `testing.B` and `b.RunParallel()` to simulate concurrency.
   - Atomic counters (`atomic.AddInt64`) are used to safely track shared state.
   - Randomized bid amounts and user IDs simulate realistic, unpredictable workloads.
   - Optional burst mode can simulate peak load by removing artificial delays between operations.

5. **Purpose**

   These benchmarks help evaluate:
   - System throughput under various contention and concurrency scenarios.
   - Latency distribution under normal and peak loads.
   - Memory consumption during high load.
   - Correctness under concurrent operations.

This performance testing framework ensures that the auction system can handle realistic loads and maintain responsiveness and consistency under concurrent operations.









