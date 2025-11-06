package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bidding-tracker/internal/biddingerrors"
	model "bidding-tracker/internal/models"
	"bidding-tracker/services/bidding/helpers"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Test RecordBidHandler
func TestRecordBidHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockBiddingServiceInterface(ctrl)
	handler := NewBiddingHandler(mockService)

	// Initialize Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/bids", handler.RecordBidHandler)

	now := time.Now().UTC()

	tests := []struct {
		name           string
		requestBody    any
		mockSetup      func()
		expectedStatus int
		expectedMsg    string
		validateData   func(t *testing.T, data map[string]any)
	}{
		{
			name: "success_valid_bid",
			requestBody: helpers.PlaceBidRequest{
				ItemID: "item1",
				UserID: "user1",
				Amount: 100,
			},
			mockSetup: func() {
				mockService.EXPECT().
					PlaceBid("item1", "user1", 100.0).
					Return(model.Bid{
						BidID:     uuid.NewString(),
						ItemID:    "item1",
						UserID:    "user1",
						Amount:    100.0,
						CreatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedMsg:    "bid recorded successfully",
			validateData: func(t *testing.T, data map[string]any) {
				bidID := data["bid_id"].(string)
				require.NotEmpty(t, bidID)
				_, parseErr := uuid.Parse(bidID)
				require.NoError(t, parseErr, "BidID should be a valid UUID")
				require.Equal(t, "item1", data["item_id"])
				require.Equal(t, "user1", data["user_id"])
				require.Equal(t, 100.0, data["amount"])
			},
		},
		{
			name:           "invalid_json",
			requestBody:    `{invalid json}`,
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid request payload",
		},
		{
			name: "missing_item_id",
			requestBody: helpers.PlaceBidRequest{
				ItemID: "",
				UserID: "user1",
				Amount: 50,
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid request payload",
		},
		{
			name: "missing_user_id",
			requestBody: helpers.PlaceBidRequest{
				ItemID: "item1",
				UserID: "",
				Amount: 50,
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid request payload",
		},
		{
			name: "invalid_amount_zero",
			requestBody: helpers.PlaceBidRequest{
				ItemID: "item1",
				UserID: "user1",
				Amount: 0,
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid request payload",
		},
		{
			name: "negative_amount",
			requestBody: helpers.PlaceBidRequest{
				ItemID: "item1",
				UserID: "user1",
				Amount: -10,
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid request payload",
		},
		{
			name: "service_bid_too_low",
			requestBody: helpers.PlaceBidRequest{
				ItemID: "item1",
				UserID: "user1",
				Amount: 50,
			},
			mockSetup: func() {
				mockService.EXPECT().
					PlaceBid("item1", "user1", 50.0).
					Return(model.Bid{}, biddingerrors.ErrBidTooLow)
			},
			expectedStatus: http.StatusConflict,
			expectedMsg:    "bid amount too low",
		},
		{
			name: "service_invalid_bid",
			requestBody: helpers.PlaceBidRequest{
				ItemID: "item1",
				UserID: "user1",
				Amount: 1, // valid for binding, service returns error
			},
			mockSetup: func() {
				mockService.EXPECT().
					PlaceBid("item1", "user1", 1.0).
					Return(model.Bid{}, biddingerrors.ErrInvalidBid)
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid bid details",
		},
		{
			name: "service_generic_error",
			requestBody: helpers.PlaceBidRequest{
				ItemID: "item1",
				UserID: "user1",
				Amount: 100,
			},
			mockSetup: func() {
				mockService.EXPECT().
					PlaceBid("item1", "user1", 100.0).
					Return(model.Bid{}, errors.New("database failure"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "internal server error",
		},
		{
			name: "extremely_large_amount",
			requestBody: helpers.PlaceBidRequest{
				ItemID: "item1",
				UserID: "user1",
				Amount: 1e18, // huge numeric value
			},
			mockSetup: func() {
				mockService.EXPECT().
					PlaceBid("item1", "user1", 1e18).
					Return(model.Bid{
						BidID:     uuid.NewString(),
						ItemID:    "item1",
						UserID:    "user1",
						Amount:    1e18,
						CreatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedMsg:    "bid recorded successfully",
			validateData: func(t *testing.T, data map[string]any) {
				bidID := data["bid_id"].(string)
				require.NotEmpty(t, bidID)
				_, parseErr := uuid.Parse(bidID)
				require.NoError(t, parseErr, "BidID should be a valid UUID")
				require.Equal(t, "item1", data["item_id"])
				require.Equal(t, "user1", data["user_id"])
				require.Equal(t, 1e18, data["amount"])
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var reqBody []byte
			var err error
			switch v := tc.requestBody.(type) {
			case string:
				reqBody = []byte(v)
			default:
				reqBody, err = json.Marshal(v)
				require.NoError(t, err)
			}

			tc.mockSetup()

			req := httptest.NewRequest(http.MethodPost, "/bids", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)

			var resp map[string]any
			err = json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			require.Contains(t, resp["message"], tc.expectedMsg)

			if tc.validateData != nil && w.Code == http.StatusCreated {
				data := resp["data"].(map[string]any)
				tc.validateData(t, data)
			}
		})
	}
}

// Test GetBidsByItemHandler
func TestGetBidsByItemHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockBiddingServiceInterface(ctrl)
	handler := NewBiddingHandler(mockService)

	// Initialize Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/items/:item_id/bids", handler.GetBidsByItemHandler)

	now := time.Now().UTC()

	tests := []struct {
		name           string
		itemID         string
		mockSetup      func()
		expectedStatus int
		expectedMsg    string
		validateData   func(t *testing.T, data []map[string]any)
	}{
		{
			name:   "success_multiple_bids",
			itemID: "item1",
			mockSetup: func() {
				mockService.EXPECT().
					GetBidsForItem("item1").
					Return([]model.Bid{
						{BidID: uuid.NewString(), ItemID: "item1", UserID: "user1", Amount: 100, CreatedAt: now},
						{BidID: uuid.NewString(), ItemID: "item1", UserID: "user2", Amount: 150, CreatedAt: now},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "bids retrieved successfully",
			validateData: func(t *testing.T, data []map[string]any) {
				require.Len(t, data, 2)
				require.Equal(t, "item1", data[0]["item_id"])
				require.Equal(t, "item1", data[1]["item_id"])
			},
		},
		{
			name:   "success_no_bids",
			itemID: "item2",
			mockSetup: func() {
				mockService.EXPECT().
					GetBidsForItem("item2").
					Return([]model.Bid{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "bids retrieved successfully",
			validateData: func(t *testing.T, data []map[string]any) {
				require.Len(t, data, 0)
			},
		},
		{
			name:   "service_no_bids_error",
			itemID: "item3",
			mockSetup: func() {
				mockService.EXPECT().
					GetBidsForItem("item3").
					Return(nil, biddingerrors.ErrNoBids)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "bids retrieved successfully",
			validateData: func(t *testing.T, data []map[string]any) {
				require.Len(t, data, 0)
			},
		},
		{
			name:   "service_generic_error",
			itemID: "item4",
			mockSetup: func() {
				mockService.EXPECT().
					GetBidsForItem("item4").
					Return(nil, errors.New("database failure"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "internal server error",
			validateData:   nil,
		},
		{
			name:   "service_nil_slice",
			itemID: "item5",
			mockSetup: func() {
				mockService.EXPECT().
					GetBidsForItem("item5").
					Return(nil, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "bids retrieved successfully",
			validateData: func(t *testing.T, data []map[string]any) {
				require.Len(t, data, 0)
			},
		},
		{
			name:   "extremely_large_number_of_bids",
			itemID: "item6",
			mockSetup: func() {
				bids := make([]model.Bid, 1000)
				for i := range bids {
					bids[i] = model.Bid{
						BidID:     uuid.NewString(),
						ItemID:    "item6",
						UserID:    fmt.Sprintf("user%d", i),
						Amount:    float64(i + 1),
						CreatedAt: now,
					}
				}
				mockService.EXPECT().GetBidsForItem("item6").Return(bids, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "bids retrieved successfully",
			validateData: func(t *testing.T, data []map[string]any) {
				require.Len(t, data, 1000)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.mockSetup()

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/items/%s/bids", tc.itemID), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)

			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			require.Contains(t, resp["message"], tc.expectedMsg)

			if tc.validateData != nil && w.Code == http.StatusOK {
				dataRaw := resp["data"].([]any)
				data := make([]map[string]any, len(dataRaw))
				for i, v := range dataRaw {
					data[i] = v.(map[string]any)
				}
				tc.validateData(t, data)
			}
		})
	}
}

// Test WinningBidHandler
func TestGetWinningBidHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockBiddingServiceInterface(ctrl)
	handler := NewBiddingHandler(mockService)

	// Initialize Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/items/:item_id/winning", handler.GetWinningBidHandler)

	now := time.Now().UTC()

	tests := []struct {
		name           string
		itemID         string
		mockSetup      func()
		expectedStatus int
		expectedMsg    string
		validateData   func(t *testing.T, data map[string]any)
	}{
		{
			name:   "success_winning_bid",
			itemID: "item1",
			mockSetup: func() {
				mockService.EXPECT().
					GetWinningBid("item1").
					Return(model.Bid{
						BidID:     uuid.NewString(),
						ItemID:    "item1",
						UserID:    "user1",
						Amount:    150.0,
						CreatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "winning bid retrieved successfully",
			validateData: func(t *testing.T, data map[string]any) {
				bidID := data["bid_id"].(string)
				require.NotEmpty(t, bidID)
				_, err := uuid.Parse(bidID)
				require.NoError(t, err, "BidID should be a valid UUID")
				require.Equal(t, "item1", data["item_id"])
				require.Equal(t, "user1", data["user_id"])
				require.Equal(t, 150.0, data["amount"])
			},
		},
		{
			name:   "no_winning_bid",
			itemID: "item2",
			mockSetup: func() {
				mockService.EXPECT().
					GetWinningBid("item2").
					Return(model.Bid{}, biddingerrors.ErrNoBids)
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "no winning bid found",
		},
		{
			name:   "service_error_generic",
			itemID: "item3",
			mockSetup: func() {
				mockService.EXPECT().
					GetWinningBid("item3").
					Return(model.Bid{}, errors.New("DB connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "internal server error",
		},
		{
			name:   "extremely_large_amount",
			itemID: "item4",
			mockSetup: func() {
				mockService.EXPECT().
					GetWinningBid("item4").
					Return(model.Bid{
						BidID:     uuid.NewString(),
						ItemID:    "item4",
						UserID:    "user_large",
						Amount:    1e12, // extremely large amount
						CreatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "winning bid retrieved successfully",
			validateData: func(t *testing.T, data map[string]any) {
				require.Equal(t, 1e12, data["amount"])
			},
		},
		{
			name:   "negative_bid_amount",
			itemID: "item1",
			mockSetup: func() {
				mockService.EXPECT().
					GetWinningBid("item1").
					Return(model.Bid{
						BidID:     uuid.NewString(),
						ItemID:    "item1",
						UserID:    "user1",
						Amount:    -100,
						CreatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "winning bid retrieved successfully",
			validateData: func(t *testing.T, data map[string]any) {
				require.Equal(t, "item1", data["item_id"])
				require.Equal(t, "user1", data["user_id"])
				require.Equal(t, -100.0, data["amount"])
			},
		},
		{
			name:   "missing_item_id",
			itemID: "",
			mockSetup: func() {
				mockService.EXPECT().
					GetWinningBid("").
					Return(model.Bid{}, fmt.Errorf("invalid item_id"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "internal server error",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.mockSetup()

			req := httptest.NewRequest(http.MethodGet, "/items/"+tc.itemID+"/winning", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)

			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			require.Contains(t, resp["message"], tc.expectedMsg)

			if tc.validateData != nil && w.Code == http.StatusOK {
				data := resp["data"].(map[string]any)
				tc.validateData(t, data)
			}
		})
	}
}

// Test GetItemsByUserHandler
func TestGetItemsByUserHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockBiddingServiceInterface(ctrl)
	handler := NewBiddingHandler(mockService)

	// Initialize Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/users/:user_id/items", handler.GetItemsByUserHandler)

	tests := []struct {
		name           string
		userID         string
		mockSetup      func()
		expectedStatus int
		expectedMsg    string
		validateData   func(t *testing.T, data []model.Item)
	}{
		{
			name:   "success_with_items",
			userID: "user1",
			mockSetup: func() {
				mockService.EXPECT().
					GetItemsByUser("user1").
					Return([]model.Item{
						{ItemID: "item1", Title: "title1", Description: "description1", StartingPrice: 50.0},
						{ItemID: "item2", Title: "title2", Description: "description2", StartingPrice: 100.0},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "items retrieved successfully",
			validateData: func(t *testing.T, data []model.Item) {
				require.Len(t, data, 2)
				require.Equal(t, "item1", data[0].ItemID)
				require.Equal(t, "title1", data[0].Title)
				require.Equal(t, "description1", data[0].Description)
				require.Equal(t, 50.0, data[0].StartingPrice)

				require.Equal(t, "item2", data[1].ItemID)
				require.Equal(t, "title2", data[1].Title)
				require.Equal(t, "description2", data[1].Description)
				require.Equal(t, 100.0, data[1].StartingPrice)
			},
		},
		{
			name:   "user_no_items",
			userID: "user2",
			mockSetup: func() {
				mockService.EXPECT().
					GetItemsByUser("user2").
					Return([]model.Item{}, biddingerrors.ErrUserNoBids)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "items retrieved successfully",
			validateData: func(t *testing.T, data []model.Item) {
				require.Len(t, data, 0)
			},
		},
		{
			name:   "service_error_generic",
			userID: "user3",
			mockSetup: func() {
				mockService.EXPECT().
					GetItemsByUser("user3").
					Return(nil, errors.New("DB connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "internal server error",
		},
		{
			name:   "missing_user_id",
			userID: "",
			mockSetup: func() {
				mockService.EXPECT().
					GetItemsByUser("").
					Return(nil, fmt.Errorf("invalid user_id"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "internal server error",
		},
		{
			name:   "extremely_large_number_of_items",
			userID: "user_large",
			mockSetup: func() {
				items := make([]model.Item, 10000)
				for i := 0; i < 10000; i++ {
					items[i] = model.Item{
						ItemID:        fmt.Sprintf("item%d", i+1),
						Title:         fmt.Sprintf("title%d", i+1),
						Description:   fmt.Sprintf("description%d", i+1),
						StartingPrice: float64(i + 1),
					}
				}
				mockService.EXPECT().
					GetItemsByUser("user_large").
					Return(items, nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "items retrieved successfully",
			validateData: func(t *testing.T, data []model.Item) {
				require.Len(t, data, 10000)
				require.Equal(t, "item1", data[0].ItemID)
				require.Equal(t, "title1", data[0].Title)
				require.Equal(t, "description1", data[0].Description)
				require.Equal(t, float64(1), data[0].StartingPrice)

				require.Equal(t, "item10000", data[9999].ItemID)
				require.Equal(t, "title10000", data[9999].Title)
				require.Equal(t, "description10000", data[9999].Description)
				require.Equal(t, float64(10000), data[9999].StartingPrice)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.mockSetup()

			req := httptest.NewRequest(http.MethodGet, "/users/"+tc.userID+"/items", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)

			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			require.Contains(t, resp["message"], tc.expectedMsg)

			if tc.validateData != nil && w.Code == http.StatusOK {
				dataBytes, _ := json.Marshal(resp["data"])
				var data []model.Item
				err := json.Unmarshal(dataBytes, &data)
				require.NoError(t, err)
				tc.validateData(t, data)
			}
		})
	}
}
