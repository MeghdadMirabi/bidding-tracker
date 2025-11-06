package perftests

import (
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	bidding "bidding-tracker/internal/biddingService"
	model "bidding-tracker/internal/models"
	repository "bidding-tracker/internal/repository"
)

// LoadScenario defines configurable benchmark parameters
type LoadScenario struct {
	Name            string
	NumUsers        int
	NumItems        int
	BidsPerUser     int
	ReadRatio       int
	MaxBidIncrement int
	Burst           bool // if true, no delay between ops
}

// OperationMetrics collects latencies safely
type OperationMetrics struct {
	latencies atomic.Value // stores []time.Duration
}

func (om *OperationMetrics) Record(d time.Duration) {
	v := om.latencies.Load()
	var l []time.Duration
	if v != nil {
		l = v.([]time.Duration)
	}
	l = append(l, d)
	om.latencies.Store(l)
}

func (om *OperationMetrics) Stats() (min, max, avg, p95, p99 time.Duration) {
	v := om.latencies.Load()
	if v == nil {
		return
	}
	latencies := v.([]time.Duration)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	min = latencies[0]
	max = latencies[len(latencies)-1]

	var total time.Duration
	for _, d := range latencies {
		total += d
	}
	avg = total / time.Duration(len(latencies))
	p95 = latencies[int(0.95*float64(len(latencies)))]
	p99 = latencies[int(0.99*float64(len(latencies)))]
	return
}

// setupRepo creates repository and bidding service with items
func setupRepo(numItems int) (*repository.MemoryRepo, *bidding.BiddingService) {
	repo := repository.NewMemoryRepo()
	svc := bidding.NewBiddingService(repo)
	for i := 0; i < numItems; i++ {
		repo.AddItem(model.Item{
			ItemID:        fmt.Sprintf("item_%d", i),
			Title:         fmt.Sprintf("title_%d", i),
			Description:   "Load test item",
			StartingPrice: 100,
		})
	}
	return repo, svc
}

// Benchmark_Load_BiddingSystem runs multiple scenarios
func Benchmark_Load_BiddingSystem(b *testing.B) {
	scenarios := []LoadScenario{
		{"Low-Contention-WriteHeavy", 200, 200, 10, 0, 50, false},
		{"High-Contention-WriteHeavy", 500, 10, 20, 0, 20, false},
		{"Mixed-Workload", 300, 50, 15, 7, 30, false},
		{"ReadHeavy", 200, 50, 5, 9, 20, false},
		{"Edge-Case-SingleItem", 100, 1, 10, 5, 10, false},
		{"Peak-Burst", 500, 50, 50, 0, 20, true},
	}

	for _, s := range scenarios {
		b.Run(s.Name, func(b *testing.B) {
			runParallelScenario(b, s)
		})
	}
}

func runParallelScenario(b *testing.B, s LoadScenario) {
	b.ReportAllocs()

	_, svc := setupRepo(s.NumItems)

	var totalOps, successfulBids, failedBids, totalReads int64
	itemSuccess := make([]int64, s.NumItems)
	metrics := &OperationMetrics{}

	start := time.Now()

	b.RunParallel(func(pb *testing.PB) {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano() + int64(time.Now().Nanosecond())))

		for pb.Next() {
			itemIndex := rnd.Intn(s.NumItems)
			itemID := fmt.Sprintf("item_%d", itemIndex)
			opType := rnd.Intn(10)

			opStart := time.Now()
			if opType < s.ReadRatio {
				_, err := svc.GetWinningBid(itemID)
				if err != nil {
					b.Logf("ignored read error: %v", err)
				}
				atomic.AddInt64(&totalReads, 1)
			} else {
				bidAmount := float64(100 + rnd.Intn(s.MaxBidIncrement))
				userID := fmt.Sprintf("user_%d", rnd.Int())
				if _, err := svc.PlaceBid(itemID, userID, bidAmount); err != nil {
					b.Logf("ignored bid error: %v", err)
					atomic.AddInt64(&failedBids, 1)
				} else {
					atomic.AddInt64(&successfulBids, 1)
					atomic.AddInt64(&itemSuccess[itemIndex], 1)
				}
			}

			metrics.Record(time.Since(opStart))
			atomic.AddInt64(&totalOps, 1)

			if !s.Burst {
				time.Sleep(time.Millisecond)
			}
		}
	})

	elapsed := time.Since(start)
	throughput := float64(totalOps) / elapsed.Seconds()
	min, max, avg, p95, p99 := metrics.Stats()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	b.Logf(
		"Scenario: %s | Items: %d | Total Ops: %d | Success Bids: %d | Failed Bids: %d | Reads: %d | Elapsed: %s | Throughput: %.2f ops/sec | Latency(us) min: %.2f avg: %.2f max: %.2f p95: %.2f p99: %.2f | Memory Alloc: %.2f MB",
		s.Name, s.NumItems, totalOps, successfulBids, failedBids, totalReads, elapsed,
		throughput,
		float64(min.Microseconds()), float64(avg.Microseconds()), float64(max.Microseconds()),
		float64(p95.Microseconds()), float64(p99.Microseconds()),
		float64(mem.Alloc)/1024/1024,
	)

	for i, v := range itemSuccess {
		if v > 0 {
			b.Logf("Item %d successful bids: %d", i, v)
		}
	}
}
