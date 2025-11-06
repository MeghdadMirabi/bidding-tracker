package server

import (
	bidding "bidding-tracker/internal/biddingService"
	handler "bidding-tracker/services/bidding/handler"

	"github.com/gin-gonic/gin"
)

// SetupRouter configures all Gin routes for the application
func SetupRouter(biddingService *bidding.BiddingService) *gin.Engine {
	router := gin.New() // New router without default middleware for full control over middleware and logging

	router.Use(gin.Recovery())          // recover from panics
	router.Use(RequestLoggerMiddleware) // custom request logging

	biddingHandler := handler.NewBiddingHandler(biddingService)

	bids := router.Group("/bids")
	{
		bids.POST("", biddingHandler.RecordBidHandler)
	}

	items := router.Group("/items")
	{
		items.GET("/:item_id/bids", biddingHandler.GetBidsByItemHandler)
		items.GET("/:item_id/winning", biddingHandler.GetWinningBidHandler)
	}

	users := router.Group("/users")
	{
		users.GET("/:user_id/items", biddingHandler.GetItemsByUserHandler)
	}

	return router
}
