// Command uss is a tiny USS simulator: it floods ES with random ingest RPCs.
//
//	# terminal 1
//	go run ./cmd/estimator
//	# terminal 2
//	go run ./cmd/uss
//
// Env (optional): GRPC_ADDR (default :9090), WORKERS (8), EVENTS (1000)
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/selseleha/go-challenge/internal/segment/adapter/grpc/gen"
)

var segments = []string{
	"sports", "music", "tech", "travel", "food", "finance", "gaming", "news",
}

func main() {
	addr := envOr("GRPC_ADDR", "localhost:9090")
	workers := envInt("WORKERS", 8)
	events := envInt("EVENTS", 1000)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()
	client := gen.NewMembershipServiceClient(conn)

	jobs := make(chan int, workers*2)
	var okCount, errCount atomic.Int64
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				userID := fmt.Sprintf("u%06d", rand.IntN(50_000))
				segment := segments[rand.IntN(len(segments))]
				_, err := client.Ingest(ctx, &gen.IngestRequest{
					UserId:  userID,
					Segment: segment,
				})
				if err != nil {
					errCount.Add(1)
					continue
				}
				okCount.Add(1)
			}
		}()
	}

	start := time.Now()
	for i := 0; i < events; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	elapsed := time.Since(start)
	log.Printf("done: ok=%d err=%d workers=%d events=%d in %s (%.0f req/s)",
		okCount.Load(), errCount.Load(), workers, events, elapsed.Round(time.Millisecond),
		float64(okCount.Load())/elapsed.Seconds(),
	)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
