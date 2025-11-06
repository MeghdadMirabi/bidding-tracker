package models

import "time"

// User represents a participant in the auction
type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// Item represents an auction item
type Item struct {
	ItemID        string  `json:"item_id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	StartingPrice float64 `json:"starting_price"`
}

// Bid represents a user's bid on an item
type Bid struct {
	BidID     string    `json:"bid_id"`
	ItemID    string    `json:"item_id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
