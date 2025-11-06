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
### Concurrency Approach

The auction system supports multiple users interacting concurrently. Concurrency control is handled primarily in the **repository layer**, while the **service layer** orchestrates business logic and the **handler layer** exposes HTTP endpoints. The Gin framework spawns a separate goroutine for each incoming HTTP request, allowing concurrent access to the system.


#### Repository Layer (`MemoryRepo`)

The repository stores all shared auction data in memory and uses a **read/write mutex (`sync.RWMutex`)** to prevent race conditions.  

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
  - Returns the bids in chronological order.  

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





