package perftests

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	bidding "bidding-tracker/internal/biddingService"
	model "bidding-tracker/internal/models"
	repository "bidding-tracker/internal/repository"
)

// Benchmark 1: PlaceBid - Isolated Items (Low Contention - Micro Benchmark)
func Benchmark_PlaceBid_Isolated(b *testing.B) {
	repo := repository.NewMemoryRepo()
	svc := bidding.NewBiddingService(repo)

	for i := 0; i < b.N; i++ {
		item := model.Item{
			ItemID:        fmt.Sprintf("item_%d", i),
			Title:         fmt.Sprintf("Low-Contention Item%d", i),
			Description:   "Independent benchmark item",
			StartingPrice: 50,
		}
		repo.AddItem(item)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user_%d", i)
		itemID := fmt.Sprintf("item_%d", i)
		bidAmount := float64(50 + rand.Intn(100))
		if _, err := svc.PlaceBid(itemID, userID, bidAmount); err != nil {
			b.Fatalf("failed to place bid: %v", err)
		}
	}
}

// Benchmark 2: PlaceBid - Shared Item (High Contention - Concurrency Benchmark)

func Benchmark_PlaceBid_ConcurrentSharedItem(b *testing.B) {
	repo := repository.NewMemoryRepo()
	svc := bidding.NewBiddingService(repo)

	item := model.Item{
		ItemID:        "shared_item_1",
		Title:         "High-Contention Item",
		Description:   "Used to simulate many users bidding concurrently",
		StartingPrice: 50,
	}
	repo.AddItem(item)

	b.ReportAllocs()
	b.ResetTimer()

	var lastBid int64 = 50

	b.RunParallel(func(pb *testing.PB) {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			userID := fmt.Sprintf("user_parallel_%d", rnd.Int())

			nextBid := atomic.AddInt64(&lastBid, int64(rnd.Intn(5)+1))
			_, _ = svc.PlaceBid(item.ItemID, userID, float64(nextBid))
		}
	})
}

// Benchmark 3: GetWinningBid - Single - Threaded (Low Contention)
func Benchmark_GetWinningBid_SingleThreaded(b *testing.B) {
	repo := repository.NewMemoryRepo()
	svc := bidding.NewBiddingService(repo)

	for i := 0; i < b.N; i++ {
		item := model.Item{
			ItemID:        fmt.Sprintf("item_%d", i),
			Title:         fmt.Sprintf("Low-Contention Item%d", i),
			Description:   "Independent benchmark item",
			StartingPrice: 50,
		}
		repo.AddItem(item)

		for j := 0; j < 10; j++ {
			userID := fmt.Sprintf("user_%d_%d", i, j)
			bidAmount := float64(50 + j*10)
			_, _ = svc.PlaceBid(item.ItemID, userID, bidAmount)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		itemID := fmt.Sprintf("item_%d", i)
		if _, err := svc.GetWinningBid(itemID); err != nil {
			b.Fatalf("failed to get winning bid: %v", err)
		}
	}
}

// Benchmark 4: GetWinningBid - Concurrent (High Contention)
func Benchmark_GetWinningBid_ConcurrentSharedItem(b *testing.B) {
	repo := repository.NewMemoryRepo()
	svc := bidding.NewBiddingService(repo)

	item := model.Item{
		ItemID:        "shared_item_1",
		Title:         "High-Contention Item",
		Description:   "Used to simulate many users reading concurrently",
		StartingPrice: 50,
	}
	repo.AddItem(item)

	for j := 0; j < 100; j++ {
		userID := fmt.Sprintf("user_%d", j)
		bidAmount := float64(50 + j)
		_, _ = svc.PlaceBid(item.ItemID, userID, bidAmount)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var counter int64

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := svc.GetWinningBid(item.ItemID); err != nil {
				b.Fatalf("failed to get winning bid: %v", err)
			}
			atomic.AddInt64(&counter, 1)
		}
	})
}

// Benchmark 5: Mixed Workload (Readers + Writers concurrently)
func Benchmark_MixedWorkload_SharedItem(b *testing.B) {
	repo := repository.NewMemoryRepo()
	svc := bidding.NewBiddingService(repo)

	item := model.Item{
		ItemID:        "shared_item_1",
		Title:         "Shared Item",
		Description:   "Used for mixed workload benchmarking",
		StartingPrice: 50,
	}
	repo.AddItem(item)

	for j := 0; j < 50; j++ {
		userID := fmt.Sprintf("user_seed_%d", j)
		bidAmount := float64(50 + j*2)
		_, _ = svc.PlaceBid(item.ItemID, userID, bidAmount)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var lastBid int64 = 150
	var counter int64

	// Ratio: 70% readers, 30% writers
	b.RunParallel(func(pb *testing.PB) {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			opType := rnd.Intn(10)
			switch {
			case opType < 3:
				// Writer: Place a new bid
				userID := fmt.Sprintf("user_writer_%d", rnd.Int())
				nextBid := atomic.AddInt64(&lastBid, int64(rnd.Intn(5)+1))
				_, _ = svc.PlaceBid(item.ItemID, userID, float64(nextBid))
			default:
				// Reader: Get winning bid
				if _, _ = svc.GetWinningBid(item.ItemID); false {
					b.Fatalf("read error") // never happens
				}
			}
			atomic.AddInt64(&counter, 1)
		}
	})
}
