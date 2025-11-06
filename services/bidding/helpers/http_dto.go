package helpers

// Request/Response DTOs
type PlaceBidRequest struct {
	ItemID string  `json:"item_id" binding:"required"`
	UserID string  `json:"user_id" binding:"required"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

type BidResponse struct {
	BidID     string  `json:"bid_id"`
	ItemID    string  `json:"item_id"`
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	CreatedAt string  `json:"created_at"`
}
