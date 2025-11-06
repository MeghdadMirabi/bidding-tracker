package repository

import (
	"bidding-tracker/internal/biddingerrors"
	model "bidding-tracker/internal/models"
	"fmt"
	"sync"
)

// AuctionDB defines the bid storage interface for the auction system
type AuctionDB interface {
	RecordBidForItem(bid model.Bid) error
	GetBidsByItem(itemID string) ([]model.Bid, error)
	GetWinningBid(itemID string) (model.Bid, error)
	GetItemsByUser(userID string) ([]model.Item, error)
}

// MemoryRepo is a concurrency-safe in-memory implementation of AuctionDB
type MemoryRepo struct {
	mu        sync.RWMutex
	bids      map[string][]model.Bid // key: itemID -> value: list of bids
	items     map[string]model.Item  // key: itemID -> value: item
	userItems map[string][]string    // key: userID -> value: list of itemIDs user has bid on
}

// NewMemoryRepo creates a new in-memory repository instance
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		bids:      make(map[string][]model.Bid),
		items:     make(map[string]model.Item),
		userItems: make(map[string][]string),
	}
}

// RecordBidForItem records a user's bid on an item
func (r *MemoryRepo) RecordBidForItem(bid model.Bid) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[bid.ItemID]; !ok {
		return fmt.Errorf("record bid for item %s: %w", bid.ItemID, biddingerrors.ErrItemNotFound)
	}

	r.bids[bid.ItemID] = append(r.bids[bid.ItemID], bid)

	for _, id := range r.userItems[bid.UserID] {
		if id == bid.ItemID {
			return nil
		}
	}
	r.userItems[bid.UserID] = append(r.userItems[bid.UserID], bid.ItemID)

	return nil
}

// GetBidsByItem returns all bids for an item
func (r *MemoryRepo) GetBidsByItem(itemID string) ([]model.Bid, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bids, ok := r.bids[itemID]
	if !ok || len(bids) == 0 {
		return nil, fmt.Errorf("get bids for item %s: %w", itemID, biddingerrors.ErrNoBids)
	}
	return append([]model.Bid(nil), bids...), nil
}

// GetWinningBid returns the highest bid for an item
func (r *MemoryRepo) GetWinningBid(itemID string) (model.Bid, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bids, ok := r.bids[itemID]
	if !ok || len(bids) == 0 {
		return model.Bid{}, fmt.Errorf("get winning bid for item %s: %w", itemID, biddingerrors.ErrNoBids)
	}

	winning := bids[0]
	for _, b := range bids[1:] {
		if b.Amount > winning.Amount || (b.Amount == winning.Amount && b.CreatedAt.Before(winning.CreatedAt)) {
			winning = b
		}
	}
	return winning, nil
}

// GetItemsByUser returns all items a user has bid on
func (r *MemoryRepo) GetItemsByUser(userID string) ([]model.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	itemIDs, ok := r.userItems[userID]
	if !ok || len(itemIDs) == 0 {
		return nil, fmt.Errorf("get items for user %s: %w", userID, biddingerrors.ErrUserNoBids)
	}

	items := make([]model.Item, 0, len(itemIDs))
	for _, id := range itemIDs {
		if item, exists := r.items[id]; exists {
			items = append(items, item)
		}
	}
	return items, nil
}

// AddItem adds an item to the repository. This method is intended for tests only.
func (r *MemoryRepo) AddItem(item model.Item) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.ItemID] = item
}
