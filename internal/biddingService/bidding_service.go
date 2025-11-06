package bidding

import (
	"bidding-tracker/internal/biddingerrors"
	"bidding-tracker/internal/models"
	"bidding-tracker/internal/repository"
	"bidding-tracker/utils"
	"errors"
	"fmt"
	"time"
)

// BiddingService defines the business logic for auction bidding
type BiddingService struct {
	repo repository.AuctionDB
}

// NewBiddingService creates a new BiddingService instance
func NewBiddingService(repo repository.AuctionDB) *BiddingService {
	return &BiddingService{
		repo: repo,
	}
}

// PlaceBid validates and records a user's bid for an item
func (s *BiddingService) PlaceBid(itemID, userID string, amount float64) (models.Bid, error) {
	if err := s.validateBid(itemID, userID, amount); err != nil {
		return models.Bid{}, err
	}

	bid := models.Bid{
		BidID:     utils.GenerateID(),
		ItemID:    itemID,
		UserID:    userID,
		Amount:    amount,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.RecordBidForItem(bid); err != nil {
		return models.Bid{}, fmt.Errorf("service: failed to record bid for item %s by user %s: %w", itemID, userID, err)
	}

	return bid, nil
}

// validateBid checks input validity and business rules for bidding
func (s *BiddingService) validateBid(itemID, userID string, amount float64) error {
	if itemID == "" || userID == "" {
		return fmt.Errorf("service: %w - missing itemID or userID", biddingerrors.ErrInvalidBid)
	}
	if amount <= 0 {
		return fmt.Errorf("service: %w - non-positive bid amount", biddingerrors.ErrInvalidBid)
	}

	winningBid, err := s.repo.GetWinningBid(itemID)
	if err == nil {
		if amount <= winningBid.Amount {
			return fmt.Errorf("service: %w - current highest bid is %.2f", biddingerrors.ErrBidTooLow, winningBid.Amount)
		}
	} else if !errors.Is(err, biddingerrors.ErrNoBids) {
		return fmt.Errorf("service: failed to check winning bid: %w", err)
	}

	return nil
}

// GetBidsForItem returns all bids for a specific item
func (s *BiddingService) GetBidsForItem(itemID string) ([]models.Bid, error) {
	if itemID == "" {
		return nil, fmt.Errorf("service: %w - empty item ID", biddingerrors.ErrInvalidBid)
	}

	bids, err := s.repo.GetBidsByItem(itemID)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get bids for item %s: %w", itemID, err)
	}

	return bids, nil
}

// GetWinningBid returns the highest bid for a specific item
func (s *BiddingService) GetWinningBid(itemID string) (models.Bid, error) {
	if itemID == "" {
		return models.Bid{}, fmt.Errorf("service: %w - empty item ID", biddingerrors.ErrInvalidBid)
	}

	winningBid, err := s.repo.GetWinningBid(itemID)
	if err != nil {
		return models.Bid{}, fmt.Errorf("service: failed to get winning bid for item %s: %w", itemID, err)
	}

	return winningBid, nil
}

// GetItemsByUser returns all items a user has placed bids on
func (s *BiddingService) GetItemsByUser(userID string) ([]models.Item, error) {
	if userID == "" {
		return nil, fmt.Errorf("service: %w - empty user ID", biddingerrors.ErrInvalidBid)
	}

	items, err := s.repo.GetItemsByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get items for user %s: %w", userID, err)
	}

	return items, nil
}
