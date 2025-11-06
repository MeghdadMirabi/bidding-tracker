package repository

import (
	model "bidding-tracker/internal/models"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Helper to create a new Item
func newItem(itemID, title string, startingPrice float64) model.Item {
	return model.Item{
		ItemID:        itemID,
		Title:         title,
		Description:   fmt.Sprintf("%s description", title),
		StartingPrice: startingPrice,
	}
}

// Helper to create a new Bid
func newBid(bidID, itemID, userID string, amount float64, createdAt time.Time) model.Bid {
	return model.Bid{
		BidID:     bidID,
		ItemID:    itemID,
		UserID:    userID,
		Amount:    amount,
		CreatedAt: createdAt,
	}
}

// Test RecordBidForItem
func TestMemoryRepo_RecordBidForItem(t *testing.T) {
	t.Parallel() // Allow running in parallel with other test functions

	// Initialize repo and seed with an item
	repo := NewMemoryRepo()
	repo.items["item1"] = newItem("item1", "Item 1", 50)

	// Table-driven test cases
	tests := []struct {
		name      string
		bid       model.Bid
		wantError bool
	}{
		{name: "valid_bid", bid: newBid("bid1", "item1", "user1", 100, time.Now()), wantError: false},
		{name: "item_not_found", bid: newBid("bid2", "itemX", "user1", 50, time.Now()), wantError: true},
		{name: "bid_with_zero_amount", bid: newBid("bid3", "item1", "user2", 0, time.Now()), wantError: false},
		{name: "bid_with_negative_amount", bid: newBid("bid4", "item1", "user2", -10, time.Now()), wantError: false},
		{name: "bid_with_max_float", bid: newBid("bid5", "item1", "user3", math.MaxFloat64, time.Now()), wantError: false},
		{name: "bid_with_past_timestamp", bid: newBid("bid6", "item1", "user4", 120, time.Now().Add(-24*time.Hour)), wantError: false},
		{name: "bid_with_future_timestamp", bid: newBid("bid7", "item1", "user5", 130, time.Now().Add(24*time.Hour)), wantError: false},
		{name: "empty_itemID", bid: newBid("bid-empty", "", "userY", 100, time.Now()), wantError: true},
		{name: "empty_userID", bid: newBid("bid-empty-user", "item1", "", 120, time.Now()), wantError: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run table test cases in parallel

			err := repo.RecordBidForItem(tc.bid)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if repo.items != nil {
					bids, err := repo.GetBidsByItem(tc.bid.ItemID)
					require.NoError(t, err)
					require.Contains(t, bids, tc.bid)
				}
			}
		})
	}

	// Special case: user placing the same bid twice (idempotent behavior)
	t.Run("user_already_bid_on_same_item", func(t *testing.T) {
		bid := newBid("bid-existing", "item1", "userX", 300, time.Now())
		require.NoError(t, repo.RecordBidForItem(bid))
		// Record the same bid again
		require.NoError(t, repo.RecordBidForItem(bid))
	})

	// concurrency test
	t.Run("concurrent_bids", func(t *testing.T) {
		t.Parallel() // Run concurrency test in parallel

		// Initialize repo and seed with an item
		repo := NewMemoryRepo()
		repo.items["item1"] = newItem("item1", "Item 1", 50)

		var wg sync.WaitGroup
		concurrentCount := 50

		for i := 0; i < concurrentCount; i++ {
			wg.Add(1)
			i := i
			go func() {
				defer wg.Done()
				b := newBid(fmt.Sprintf("bid-%d", i), "item1", fmt.Sprintf("user-%d", i), float64(100+i), time.Now())
				require.NoError(t, repo.RecordBidForItem(b))
			}()
		}

		wg.Wait()

		bids, err := repo.GetBidsByItem("item1")
		require.NoError(t, err)
		require.Len(t, bids, concurrentCount)
	})
}

// Test GetBidsByItem
func TestMemoryRepo_GetBidsByItem(t *testing.T) {
	t.Parallel() // Allow running in parallel with other test functions

	// Initialize repo and seed with 4 items
	repo := NewMemoryRepo()
	repo.items["item1"] = newItem("item1", "Item 1", 50)
	repo.items["item2"] = newItem("item2", "Item 2", 75)
	repo.items["item3"] = newItem("item3", "Item 3", 100) // for large number of bids
	repo.items["item4"] = newItem("item4", "Item 4", 200) // for extreme bid amounts

	// Seed normal bids and check errors in setup
	bid1 := newBid("bid1", "item1", "user1", 100, time.Now())
	bid2 := newBid("bid2", "item1", "user2", 150, time.Now())
	require.NoError(t, repo.RecordBidForItem(bid1))
	require.NoError(t, repo.RecordBidForItem(bid2))

	// Seed large number of bids for performance/internal slice growth
	var largeBids []model.Bid
	for i := 0; i < 1000; i++ {
		b := newBid(fmt.Sprintf("bid-large-%d", i), "item3", fmt.Sprintf("user-%d", i), float64(100+i), time.Now())
		require.NoError(t, repo.RecordBidForItem(b))
		largeBids = append(largeBids, b)
	}

	// Seed bids with extreme amounts
	bidHigh := newBid("bid-high", "item4", "user-high", math.MaxFloat64, time.Now())
	bidLow := newBid("bid-low", "item4", "user-low", -math.MaxFloat64, time.Now())
	require.NoError(t, repo.RecordBidForItem(bidHigh))
	require.NoError(t, repo.RecordBidForItem(bidLow))

	// Table-driven test cases
	tests := []struct {
		name      string
		itemID    string
		wantBids  []model.Bid
		wantError bool
	}{
		{name: "existing_item_with_bids", itemID: "item1", wantBids: []model.Bid{bid1, bid2}, wantError: false},
		{name: "existing_item_no_bids", itemID: "item2", wantBids: []model.Bid{}, wantError: true}, // keep as error
		{name: "non_existing_item", itemID: "itemX", wantBids: nil, wantError: true},
		{name: "item_with_large_number_of_bids", itemID: "item3", wantBids: largeBids, wantError: false},
		{name: "item_with_extreme_bid_amounts", itemID: "item4", wantBids: []model.Bid{bidHigh, bidLow}, wantError: false},
		{name: "empty_itemID", itemID: "", wantBids: nil, wantError: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run table test cases in parallel

			bids, err := repo.GetBidsByItem(tc.itemID)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, bids, tc.wantBids)
			}
		})
	}

	// Concurrent read test
	t.Run("concurrent_reads", func(t *testing.T) {
		t.Parallel() // Run concurrent read test in parallel

		var wg sync.WaitGroup
		readCount := 50

		for i := 0; i < readCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				bids, err := repo.GetBidsByItem("item1")
				require.NoError(t, err)
				require.ElementsMatch(t, bids, []model.Bid{bid1, bid2})
			}()
		}

		wg.Wait()
	})
}

// Test GetWinningBid
func TestMemoryRepo_GetWinningBid(t *testing.T) {
	t.Parallel() // Allow running in parallel with other test functions

	// Initialize repo and seed with 4 items
	repo := NewMemoryRepo()
	repo.items["item1"] = newItem("item1", "Item 1", 50)
	repo.items["item2"] = newItem("item2", "Item 2", 75)
	repo.items["item3"] = newItem("item3", "Item 3", 100) // for large number of bids
	repo.items["item4"] = newItem("item4", "Item 4", 200) // for extreme bid amounts
	repo.items["item5"] = newItem("item5", "Item 5", 150) // for tie bids

	// Seed normal bids
	bid1 := newBid("bid1", "item1", "user1", 100, time.Now())
	bid2 := newBid("bid2", "item1", "user2", 150, time.Now())
	require.NoError(t, repo.RecordBidForItem(bid1))
	require.NoError(t, repo.RecordBidForItem(bid2))

	// Seed large number of bids
	var largeBids []model.Bid
	for i := 0; i < 1000; i++ {
		b := newBid(fmt.Sprintf("bid-large-%d", i), "item3", fmt.Sprintf("user-%d", i), float64(100+i), time.Now())
		require.NoError(t, repo.RecordBidForItem(b))
		largeBids = append(largeBids, b)
	}

	// Seed bids with extreme amounts
	bidHigh := newBid("bid-high", "item4", "user-high", math.MaxFloat64, time.Now())
	bidLow := newBid("bid-low", "item4", "user-low", -math.MaxFloat64, time.Now())
	require.NoError(t, repo.RecordBidForItem(bidHigh))
	require.NoError(t, repo.RecordBidForItem(bidLow))

	// Tie bids
	bidTie1 := newBid("bid-tie1", "item5", "userA", 200, time.Now())
	bidTie2 := newBid("bid-tie2", "item5", "userB", 200, time.Now())
	require.NoError(t, repo.RecordBidForItem(bidTie1))
	require.NoError(t, repo.RecordBidForItem(bidTie2))

	// Table-driven test cases
	tests := []struct {
		name      string
		itemID    string
		wantBid   model.Bid
		wantError bool
	}{
		{name: "existing_item_with_bids", itemID: "item1", wantBid: bid2, wantError: false},
		{name: "existing_item_no_bids", itemID: "item2", wantBid: model.Bid{}, wantError: true},
		{name: "non_existing_item", itemID: "itemX", wantBid: model.Bid{}, wantError: true},
		{name: "item_with_large_number_of_bids", itemID: "item3", wantBid: largeBids[len(largeBids)-1], wantError: false},
		{name: "item_with_extreme_bid_amounts", itemID: "item4", wantBid: bidHigh, wantError: false},
		{name: "tie_bids_first_wins", itemID: "item5", wantBid: bidTie1, wantError: false},
		{name: "empty_itemID", itemID: "", wantBid: model.Bid{}, wantError: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run table test cases in parallel

			bid, err := repo.GetWinningBid(tc.itemID)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantBid, bid)
			}
		})
	}

	// Concurrent winning bid retrieval test
	t.Run("concurrent_get_winning_bid", func(t *testing.T) {
		t.Parallel() // Run concurrent test in parallel

		var wg sync.WaitGroup
		readCount := 50

		for i := 0; i < readCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				bid, err := repo.GetWinningBid("item1")
				require.NoError(t, err)
				require.Equal(t, bid2, bid)
			}()
		}

		wg.Wait()
	})
}

// Test GetItemsByUser
func TestMemoryRepo_GetItemsByUser(t *testing.T) {
	t.Parallel() // Allow running in parallel with other test functions

	// Initialize repo and seed with 4 items
	repo := NewMemoryRepo()
	repo.items["item1"] = newItem("item1", "Item 1", 50)
	repo.items["item2"] = newItem("item2", "Item 2", 75)
	repo.items["item3"] = newItem("item3", "Item 3", 100) // for large number of bids
	repo.items["item4"] = newItem("item4", "Item 4", 200) // for extreme bid amounts
	repo.items["item5"] = newItem("item5", "Item 5", 250) // for duplicates

	// Seed bids
	bid1 := newBid("bid1", "item1", "user1", 100, time.Now())
	bid2 := newBid("bid2", "item2", "user1", 150, time.Now())
	bid3 := newBid("bid3", "item3", "user2", 200, time.Now())
	bid4 := newBid("bid4", "item4", "user3", 250, time.Now())
	bid5 := newBid("bid5", "item5", "user6", 300, time.Now())
	require.NoError(t, repo.RecordBidForItem(bid1))
	require.NoError(t, repo.RecordBidForItem(bid2))
	require.NoError(t, repo.RecordBidForItem(bid3))
	require.NoError(t, repo.RecordBidForItem(bid4))
	require.NoError(t, repo.RecordBidForItem(bid5))

	// Seed large number of bids for user4
	for i := 0; i < 1000; i++ {
		b := newBid(fmt.Sprintf("bid-large-%d", i), "item3", "user4", float64(100+i), time.Now())
		require.NoError(t, repo.RecordBidForItem(b))
	}

	// Duplicate bids for same item for user6
	require.NoError(t, repo.RecordBidForItem(newBid("bid6", "item5", "user6", 350, time.Now())))

	// Table-driven test cases
	tests := []struct {
		name      string
		userID    string
		wantItems []model.Item
		wantError bool
	}{
		{name: "user_with_multiple_items", userID: "user1", wantItems: []model.Item{repo.items["item1"], repo.items["item2"]}, wantError: false},
		{name: "user_with_single_item", userID: "user2", wantItems: []model.Item{repo.items["item3"]}, wantError: false},
		{name: "user_with_no_items", userID: "userX", wantItems: nil, wantError: true},
		{name: "user_with_large_number_of_items", userID: "user4", wantItems: []model.Item{repo.items["item3"]}, wantError: false},
		{name: "user_with_extreme_bid_amounts", userID: "user3", wantItems: []model.Item{repo.items["item4"]}, wantError: false},
		{name: "empty_userID", userID: "", wantItems: nil, wantError: true},
		{name: "duplicate_bids_same_item", userID: "user6", wantItems: []model.Item{repo.items["item5"]}, wantError: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run table test cases in parallel

			items, err := repo.GetItemsByUser(tc.userID)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, items, tc.wantItems)
			}
		})
	}

	// Concurrent read test
	t.Run("concurrent_get_items_by_user", func(t *testing.T) {
		t.Parallel() // Run concurrent test in parallel

		var wg sync.WaitGroup
		readCount := 50

		for i := 0; i < readCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				items, err := repo.GetItemsByUser("user1")
				require.NoError(t, err)
				require.ElementsMatch(t, items, []model.Item{repo.items["item1"], repo.items["item2"]})
			}()
		}

		wg.Wait()
	})
}
