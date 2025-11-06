package integrationtests

import (
	"net/http"
	"testing"
	"time"

	model "bidding-tracker/internal/models"
	"bidding-tracker/services/bidding/helpers"

	"github.com/stretchr/testify/require"
)

// RecordBidHandler Tests
func TestRecordBidHandler(t *testing.T) {
	tests := []struct {
		name       string
		item       model.Item
		request    any
		wantStatus int
	}{
		{
			name: "Valid_Bid",
			item: model.Item{
				ItemID:        "item1",
				Title:         "title1",
				Description:   "description1",
				StartingPrice: 50.0,
			},
			request: helpers.PlaceBidRequest{
				ItemID: "item1",
				UserID: "user1",
				Amount: 100,
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "Invalid_JSON",
			item:       model.Item{},
			request:    "{item_id: 'missing quotes', amount: 100}", // invalid JSON
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := SetupTestRouterWithItems(tt.item)
			resp, w := ExecuteRequestAndParse(t, router, http.MethodPost, "/bids", tt.request)
			require.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusCreated {
				require.Equal(t, "item1", resp["item_id"])
				require.Equal(t, "user1", resp["user_id"])
				require.Equal(t, 100.0, resp["amount"])
				require.NotEmpty(t, resp["bid_id"])

				_, err := time.Parse(time.RFC3339, resp["created_at"].(string))
				require.NoError(t, err)
			}
		})
	}
}

// GetBidsByItemHandler Tests
func TestGetBidsByItemHandler(t *testing.T) {
	tests := []struct {
		name       string
		items      []model.Item
		seedBids   []helpers.PlaceBidRequest
		itemID     string
		wantCount  int
		wantStatus int
	}{
		{
			name:       "With_Bids",
			items:      []model.Item{{ItemID: "item1", Title: "title1", Description: "description1", StartingPrice: 50}},
			seedBids:   []helpers.PlaceBidRequest{{ItemID: "item1", UserID: "user1", Amount: 100}},
			itemID:     "item1",
			wantCount:  1,
			wantStatus: http.StatusOK,
		},
		{
			name:       "No_Bids",
			items:      []model.Item{{ItemID: "item2", Title: "title2", Description: "description2", StartingPrice: 30}},
			itemID:     "item2",
			wantCount:  0,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Item_Not_Found",
			items:      []model.Item{},
			itemID:     "nonexistent",
			wantCount:  0,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := SetupTestRouterWithItems(tt.items...)
			for _, bid := range tt.seedBids {
				_, w := ExecuteRequestAndParse(t, router, http.MethodPost, "/bids", bid)
				require.Equal(t, http.StatusCreated, w.Code)
			}

			resp, w := ExecuteRequestAndParse(t, router, http.MethodGet, "/items/"+tt.itemID+"/bids", nil)
			require.Equal(t, tt.wantStatus, w.Code)

			bids := resp["data"].([]any)
			require.Len(t, bids, tt.wantCount)
		})
	}
}

// GetWinningBidHandler Tests
func TestGetWinningBidHandler(t *testing.T) {
	tests := []struct {
		name       string
		items      []model.Item
		seedBids   []helpers.PlaceBidRequest
		itemID     string
		wantUser   string
		wantAmount float64
		wantStatus int
	}{
		{
			name:  "With_Bids",
			items: []model.Item{{ItemID: "item1", Title: "title1", Description: "description1", StartingPrice: 50}},
			seedBids: []helpers.PlaceBidRequest{
				{ItemID: "item1", UserID: "user1", Amount: 100},
				{ItemID: "item1", UserID: "user3", Amount: 120},
				{ItemID: "item1", UserID: "user2", Amount: 150},
			},
			itemID:     "item1",
			wantUser:   "user2",
			wantAmount: 150,
			wantStatus: http.StatusOK,
		},
		{
			name:       "No_Bids",
			items:      []model.Item{{ItemID: "item2", Title: "title2", Description: "description2", StartingPrice: 30}},
			itemID:     "item2",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Item_Not_Found",
			items:      []model.Item{},
			itemID:     "nonexistent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := SetupTestRouterWithItems(tt.items...)
			for _, bid := range tt.seedBids {
				_, w := ExecuteRequestAndParse(t, router, http.MethodPost, "/bids", bid)
				require.Equal(t, http.StatusCreated, w.Code)
			}

			resp, w := ExecuteRequestAndParse(t, router, http.MethodGet, "/items/"+tt.itemID+"/winning", nil)
			require.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				data := resp["data"].(map[string]any)
				require.Equal(t, tt.itemID, data["item_id"])
				require.Equal(t, tt.wantUser, data["user_id"])
				require.Equal(t, tt.wantAmount, data["amount"])
				_, err := time.Parse(time.RFC3339, data["created_at"].(string))
				require.NoError(t, err)
			}
		})
	}
}

// GetItemsByUserHandler Tests
func TestGetItemsByUserHandler(t *testing.T) {
	router := SetupTestRouterWithItems(
		model.Item{ItemID: "item1", Title: "title1", Description: "description1", StartingPrice: 50},
		model.Item{ItemID: "item2", Title: "title2", Description: "description2", StartingPrice: 30},
	)

	// Seed bids
	bids := []helpers.PlaceBidRequest{
		{ItemID: "item1", UserID: "user1", Amount: 100},
		{ItemID: "item2", UserID: "user1", Amount: 200},
	}
	for _, bid := range bids {
		_, w := ExecuteRequestAndParse(t, router, http.MethodPost, "/bids", bid)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	tests := []struct {
		name            string
		userID          string
		expectedItemIDs []string
	}{
		{
			name:            "User_With_Items",
			userID:          "user1",
			expectedItemIDs: []string{"item1", "item2"},
		},
		{
			name:            "UserWithNoItems",
			userID:          "user2",
			expectedItemIDs: []string{},
		},
		{
			name:            "NonexistentUser",
			userID:          "nonexistent",
			expectedItemIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, w := ExecuteRequestAndParse(t, router, http.MethodGet, "/users/"+tt.userID+"/items", nil)
			require.Equal(t, http.StatusOK, w.Code)

			items := resp["data"].([]any)
			require.Len(t, items, len(tt.expectedItemIDs))

			itemIDs := map[string]bool{}
			for _, i := range items {
				it := i.(map[string]any)
				itemIDs[it["item_id"].(string)] = true
			}
			for _, id := range tt.expectedItemIDs {
				require.True(t, itemIDs[id])
			}
		})
	}
}
