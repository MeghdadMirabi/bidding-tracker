package helpers

import (
	"errors"
	"fmt"
	"net/http"

	"bidding-tracker/internal/biddingerrors"
	"bidding-tracker/utils"

	"github.com/gin-gonic/gin"
)

// HandleBindError sends a standardized JSON error for binding failures
func HandleBindError(c *gin.Context, handlerName string, err error) {
	wrappedErr := fmt.Errorf("invalid request payload: %w", err)
	utils.JSONError(c, http.StatusBadRequest, wrappedErr, "invalid request payload")
	utils.Warn(handlerName+": binding error", map[string]any{"error": err.Error()})
}

// MapErrorToHTTP maps domain/service errors to HTTP status code and message
func MapErrorToHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, biddingerrors.ErrItemNotFound):
		return http.StatusNotFound, "item not found"
	case errors.Is(err, biddingerrors.ErrInvalidBid):
		return http.StatusBadRequest, "invalid bid details"
	case errors.Is(err, biddingerrors.ErrBidTooLow):
		return http.StatusConflict, "bid amount too low"
	case errors.Is(err, biddingerrors.ErrNoBids):
		return http.StatusOK, "no bids found for item"
	case errors.Is(err, biddingerrors.ErrUserNoBids):
		return http.StatusOK, "no items found for user"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

// LogSuccess is a small helper to standardize logging of successful operations
func LogSuccess(handlerName, message string, ctx map[string]any) {
	utils.Info(handlerName+": "+message, ctx)
}
