package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"bidding-tracker/internal/biddingerrors"
	model "bidding-tracker/internal/models"
	"bidding-tracker/services/bidding/helpers"
	"bidding-tracker/utils"

	"github.com/gin-gonic/gin"
)

type BiddingServiceInterface interface {
	PlaceBid(itemID, userID string, amount float64) (model.Bid, error)
	GetBidsForItem(itemID string) ([]model.Bid, error)
	GetWinningBid(itemID string) (model.Bid, error)
	GetItemsByUser(userID string) ([]model.Item, error)
}

type BiddingHandler struct {
	service BiddingServiceInterface
}

func NewBiddingHandler(service BiddingServiceInterface) *BiddingHandler {
	return &BiddingHandler{service: service}
}

// RecordBidHandler handles POST /bids
func (h *BiddingHandler) RecordBidHandler(c *gin.Context) {
	var req helpers.PlaceBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.HandleBindError(c, "RecordBidHandler", err)
		return
	}

	bid, err := h.service.PlaceBid(req.ItemID, req.UserID, req.Amount)
	if err != nil {
		status, message := helpers.MapErrorToHTTP(err)
		utils.JSONError(c, status, fmt.Errorf("%s: %w", message, err), message)
		utils.Error("RecordBidHandler: failed to record bid", map[string]any{
			"handler": "RecordBidHandler",
			"item_id": req.ItemID,
			"user_id": req.UserID,
			"error":   err.Error(),
		})
		return
	}

	resp := helpers.BidResponse{
		BidID:     bid.BidID,
		ItemID:    bid.ItemID,
		UserID:    bid.UserID,
		Amount:    bid.Amount,
		CreatedAt: bid.CreatedAt.UTC().Format(time.RFC3339),
	}

	utils.JSONResponse(c, http.StatusCreated, resp, "bid recorded successfully")
	helpers.LogSuccess("RecordBidHandler", "bid recorded successfully", map[string]any{
		"bid_id":  bid.BidID,
		"item_id": bid.ItemID,
		"user_id": req.UserID,
		"amount":  bid.Amount,
	})
}

// GetBidsByItemHandler handles GET /items/:item_id/bids
func (h *BiddingHandler) GetBidsByItemHandler(c *gin.Context) {
	itemID := c.Param("item_id")
	bids, err := h.service.GetBidsForItem(itemID)
	if err != nil && !errors.Is(err, biddingerrors.ErrNoBids) {
		status, message := helpers.MapErrorToHTTP(err)
		utils.JSONError(c, status, fmt.Errorf("%s: %w", message, err), message)
		utils.Warn("GetBidsByItemHandler: error retrieving bids", map[string]any{"item_id": itemID, "error": err.Error()})
		return
	}

	if bids == nil {
		bids = []model.Bid{}
	}

	utils.JSONResponse(c, http.StatusOK, bids, "bids retrieved successfully")
	helpers.LogSuccess("GetBidsByItemHandler", "bids retrieved successfully", map[string]any{
		"item_id": itemID,
		"count":   len(bids),
	})
}

// GetWinningBidHandler handles GET /items/:item_id/winning
func (h *BiddingHandler) GetWinningBidHandler(c *gin.Context) {
	itemID := c.Param("item_id")
	bid, err := h.service.GetWinningBid(itemID)
	if err != nil {
		status, message := helpers.MapErrorToHTTP(err)
		// For auction, winning bid not found -> 404
		if errors.Is(err, biddingerrors.ErrNoBids) {
			utils.JSONError(c, http.StatusNotFound, err, "no winning bid found")
			utils.Info("GetWinningBidHandler: no winning bid found", map[string]any{"item_id": itemID})
			return
		}
		utils.JSONError(c, status, fmt.Errorf("%s: %w", message, err), message)
		utils.Warn("GetWinningBidHandler: winning bid error", map[string]any{"item_id": itemID, "error": err.Error()})
		return
	}

	resp := helpers.BidResponse{
		BidID:     bid.BidID,
		ItemID:    bid.ItemID,
		UserID:    bid.UserID,
		Amount:    bid.Amount,
		CreatedAt: bid.CreatedAt.UTC().Format(time.RFC3339),
	}

	utils.JSONResponse(c, http.StatusOK, resp, "winning bid retrieved successfully")
	helpers.LogSuccess("GetWinningBidHandler", "winning bid retrieved successfully", map[string]any{
		"bid_id":  bid.BidID,
		"item_id": bid.ItemID,
		"user_id": bid.UserID,
		"amount":  bid.Amount,
	})
}

// GetItemsByUserHandler handles GET /users/:user_id/items
func (h *BiddingHandler) GetItemsByUserHandler(c *gin.Context) {
	userID := c.Param("user_id")
	items, err := h.service.GetItemsByUser(userID)
	if err != nil && !errors.Is(err, biddingerrors.ErrUserNoBids) {
		status, message := helpers.MapErrorToHTTP(err)
		utils.JSONError(c, status, fmt.Errorf("%s: %w", message, err), message)
		utils.Warn("GetItemsByUserHandler: error retrieving items", map[string]any{"user_id": userID, "error": err.Error()})
		return
	}

	if items == nil {
		items = []model.Item{}
	}

	utils.JSONResponse(c, http.StatusOK, items, "items retrieved successfully")
	helpers.LogSuccess("GetItemsByUserHandler", "items retrieved successfully", map[string]any{
		"user_id":     userID,
		"items_count": len(items),
	})
}
