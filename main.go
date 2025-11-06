package main

import (
	bidding "bidding-tracker/internal/biddingService"
	model "bidding-tracker/internal/models"
	"bidding-tracker/internal/repository"
	"bidding-tracker/internal/server"
	"fmt"
	"os"
)

func main() {

	repo := repository.NewMemoryRepo()

	prepopulateItems(repo)

	biddingSvc := bidding.NewBiddingService(repo)

	router := server.SetupRouter(biddingSvc)

	port := getPort()
	fmt.Printf("Starting auction server on %s...\n", port)
	if err := router.Run(port); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

// prepopulateItems adds sample items to the in-memory repo
func prepopulateItems(repo *repository.MemoryRepo) {
	items := []model.Item{
		{ItemID: "item1", Title: "title1", Description: "description1", StartingPrice: 100},
		{ItemID: "item2", Title: "title2", Description: "Description2", StartingPrice: 200},
		{ItemID: "item3", Title: "title3", Description: "Description3", StartingPrice: 150},
	}

	for _, item := range items {
		repo.AddItem(item)
	}
}

// getPort returns the server port from env or defaults to ":8080"
func getPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return fmt.Sprintf(":%s", p)
	}
	return ":8080"
}
