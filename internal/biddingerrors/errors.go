package biddingerrors

import "errors"

// Repository-level errors
var (
	ErrItemNotFound = errors.New("item not found")
	ErrNoBids       = errors.New("no bids found for item")
	ErrUserNoBids   = errors.New("user has not placed any bids")
)

// business logic errors
var (
	ErrInvalidBid = errors.New("invalid bid")
	ErrBidTooLow  = errors.New("bid amount too low")
)
