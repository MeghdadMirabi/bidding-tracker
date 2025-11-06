package bidding

import (
	"bidding-tracker/internal/biddingerrors"
	model "bidding-tracker/internal/models"
	"bidding-tracker/internal/repository"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Tests PlaceBid
func TestBiddingService_PlaceBid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockAuctionDB(ctrl)
	service := NewBiddingService(mockRepo)

	now := time.Now().UTC()

	// Table-driven test cases
	tests := []struct {
		name          string
		itemID        string
		userID        string
		amount        float64
		mockSetup     func()
		expectError   bool
		expectedError error
	}{
		{
			name:   "valid_first_bid",
			itemID: "item1",
			userID: "user1",
			amount: 100,
			mockSetup: func() {
				mockRepo.EXPECT().GetWinningBid("item1").Return(model.Bid{}, biddingerrors.ErrNoBids)
				mockRepo.EXPECT().RecordBidForItem(gomock.Any()).Return(nil)
			},
			expectError:   false,
			expectedError: nil,
		},
		{
			name:          "empty_itemID",
			itemID:        "",
			userID:        "user1",
			amount:        50,
			mockSetup:     func() {},
			expectError:   true,
			expectedError: biddingerrors.ErrInvalidBid,
		},
		{
			name:          "empty_userID",
			itemID:        "item1",
			userID:        "",
			amount:        50,
			mockSetup:     func() {},
			expectError:   true,
			expectedError: biddingerrors.ErrInvalidBid,
		},
		{
			name:          "zero_amount",
			itemID:        "item1",
			userID:        "user1",
			amount:        0,
			mockSetup:     func() {},
			expectError:   true,
			expectedError: biddingerrors.ErrInvalidBid,
		},
		{
			name:          "negative_amount",
			itemID:        "item1",
			userID:        "user1",
			amount:        -50,
			mockSetup:     func() {},
			expectError:   true,
			expectedError: biddingerrors.ErrInvalidBid,
		},
		{
			name:   "bid_too_low",
			itemID: "item1",
			userID: "user2",
			amount: 80,
			mockSetup: func() {
				mockRepo.EXPECT().GetWinningBid("item1").Return(model.Bid{Amount: 100}, nil)
			},
			expectError:   true,
			expectedError: biddingerrors.ErrBidTooLow,
		},
		{
			name:   "repo_fails",
			itemID: "item1",
			userID: "user3",
			amount: 120,
			mockSetup: func() {
				mockRepo.EXPECT().GetWinningBid("item1").Return(model.Bid{Amount: 100}, nil)
				mockRepo.EXPECT().RecordBidForItem(gomock.Any()).Return(errors.New("repo write failed"))
			},
			expectError:   true,
			expectedError: nil, // Service wraps repo error, we don’t match specific error here
		},
		{
			name:   "bid_max_float",
			itemID: "item1",
			userID: "user4",
			amount: math.MaxFloat64,
			mockSetup: func() {
				mockRepo.EXPECT().GetWinningBid("item1").Return(model.Bid{Amount: 100}, nil)
				mockRepo.EXPECT().RecordBidForItem(gomock.Any()).Return(nil)
			},
			expectError:   false,
			expectedError: nil,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run tests concurrently

			tc.mockSetup()

			bid, err := service.PlaceBid(tc.itemID, tc.userID, tc.amount)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectedError != nil {
					require.True(t, errors.Is(err, tc.expectedError), "expected error: %v, got: %v", tc.expectedError, err)
				}
			} else {
				require.NoError(t, err)

				// Validate generated BidID
				require.NotEmpty(t, bid.BidID)
				_, parseErr := uuid.Parse(bid.BidID)
				require.NoError(t, parseErr, "BidID should be a valid UUID")

				// Validate bid fields
				require.Equal(t, tc.itemID, bid.ItemID)
				require.Equal(t, tc.userID, bid.UserID)
				require.Equal(t, tc.amount, bid.Amount)
				require.WithinDuration(t, now, bid.CreatedAt, 2*time.Second)
			}
		})
	}
}

// Tests GetBidsForItem
func TestBiddingService_GetBidsForItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockAuctionDB(ctrl)
	service := NewBiddingService(mockRepo)

	now := time.Now().UTC()

	// Initialize bids
	bidsExample := []model.Bid{
		{BidID: "bid1", ItemID: "item1", UserID: "user1", Amount: 100, CreatedAt: now},
		{BidID: "bid2", ItemID: "item1", UserID: "user2", Amount: 150, CreatedAt: now.Add(1 * time.Second)},
	}

	tests := []struct {
		name          string
		itemID        string
		mockSetup     func()
		expectError   bool
		expectedError error
		expectedBids  []model.Bid
	}{
		{
			name:   "valid_item_with_bids",
			itemID: "item1",
			mockSetup: func() {
				mockRepo.EXPECT().GetBidsByItem("item1").Return(bidsExample, nil)
			},
			expectError:   false,
			expectedError: nil,
			expectedBids:  bidsExample,
		},
		{
			name:   "valid_item_no_bids",
			itemID: "item2",
			mockSetup: func() {
				mockRepo.EXPECT().GetBidsByItem("item2").Return([]model.Bid{}, nil)
			},
			expectError:   false,
			expectedError: nil,
			expectedBids:  []model.Bid{},
		},
		{
			name:          "empty_itemID",
			itemID:        "",
			mockSetup:     func() {},
			expectError:   true,
			expectedError: biddingerrors.ErrInvalidBid,
		},
		{
			name:   "repo_error",
			itemID: "item3",
			mockSetup: func() {
				mockRepo.EXPECT().GetBidsByItem("item3").Return(nil, errors.New("db failure"))
			},
			expectError:   true,
			expectedError: nil, // Service wraps repo error
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run tests concurrently

			tc.mockSetup()

			bids, err := service.GetBidsForItem(tc.itemID)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectedError != nil {
					require.True(t, errors.Is(err, tc.expectedError), "expected error: %v, got: %v", tc.expectedError, err)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedBids, bids)
			}
		})
	}
}

// Test GetWinningBid
func TestBiddingService_GetWinningBid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockAuctionDB(ctrl)
	service := NewBiddingService(mockRepo)

	now := time.Now().UTC()

	// Table-driven test cases
	tests := []struct {
		name        string
		itemID      string
		mockSetup   func()
		expectError bool
	}{
		{
			name:   "valid_item_with_winning_bid",
			itemID: "item1",
			mockSetup: func() {
				mockRepo.EXPECT().GetWinningBid("item1").Return(model.Bid{
					BidID:     uuid.NewString(),
					ItemID:    "item1",
					UserID:    "user1",
					Amount:    100,
					CreatedAt: now,
				}, nil)
			},
			expectError: false,
		},
		{
			name:        "empty_itemID",
			itemID:      "",
			mockSetup:   func() {},
			expectError: true,
		},
		{
			name:   "repo_returns_no_bids",
			itemID: "item2",
			mockSetup: func() {
				mockRepo.EXPECT().GetWinningBid("item2").Return(model.Bid{}, biddingerrors.ErrNoBids)
			},
			expectError: true,
		},
		{
			name:   "repo_returns_error",
			itemID: "item3",
			mockSetup: func() {
				mockRepo.EXPECT().GetWinningBid("item3").Return(model.Bid{}, errors.New("repo error"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run tests concurrently

			tc.mockSetup()

			bid, err := service.GetWinningBid(tc.itemID)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Validate bid fields
				require.NotEmpty(t, bid.BidID)
				_, err := uuid.Parse(bid.BidID)
				require.NoError(t, err, "BidID should be a valid UUID")
				require.Equal(t, tc.itemID, bid.ItemID)
				require.Equal(t, "user1", bid.UserID)
				require.Equal(t, 100.0, bid.Amount)
				require.WithinDuration(t, now, bid.CreatedAt, 1*time.Second)
			}
		})
	}
}

// Test GetItemsByUser
func TestBiddingService_GetItemsByUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockAuctionDB(ctrl)
	service := NewBiddingService(mockRepo)

	// initialize items
	itemsExample := []model.Item{
		{
			ItemID:        "item1",
			Title:         "title1",
			Description:   "description1",
			StartingPrice: 1000.0,
		},
		{
			ItemID:        "item2",
			Title:         "title2",
			Description:   "description2",
			StartingPrice: 500.0,
		},
	}

	// Table-driven test cases
	tests := []struct {
		name          string
		userID        string
		mockSetup     func()
		expectError   bool
		expectedError error
		expectedItems []model.Item
	}{
		{
			name:   "valid_user_with_items",
			userID: "user1",
			mockSetup: func() {
				mockRepo.EXPECT().GetItemsByUser("user1").Return(itemsExample, nil)
			},
			expectError:   false,
			expectedError: nil,
			expectedItems: itemsExample,
		},
		{
			name:   "valid_user_no_items",
			userID: "user2",
			mockSetup: func() {
				mockRepo.EXPECT().GetItemsByUser("user2").Return([]model.Item{}, nil)
			},
			expectError:   false,
			expectedError: nil,
			expectedItems: []model.Item{},
		},
		{
			name:          "empty_userID",
			userID:        "",
			mockSetup:     func() {},
			expectError:   true,
			expectedError: biddingerrors.ErrInvalidBid,
			expectedItems: nil,
		},
		{
			name:   "repo_error",
			userID: "user3",
			mockSetup: func() {
				mockRepo.EXPECT().GetItemsByUser("user3").Return(nil, errors.New("db failure"))
			},
			expectError:   true,
			expectedError: nil, // Service wraps repo error
			expectedItems: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run tests concurrently

			tc.mockSetup()

			items, err := service.GetItemsByUser(tc.userID)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectedError != nil {
					require.True(t, errors.Is(err, tc.expectedError),
						"expected error: %v, got: %v", tc.expectedError, err)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedItems, items)
			}
		})
	}
}
