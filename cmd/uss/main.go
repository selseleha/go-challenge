// Command uss is a tiny USS simulator: it floods ES with random ingest RPCs.
//
//	# terminal 1
//	go run ./cmd/estimator
//	# terminal 2
//	go run ./cmd/uss
//
// Settings come from .env / environment (see .env.example).
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/selseleha/go-challenge/internal/config"
	"github.com/selseleha/go-challenge/internal/segment/adapter/grpc/gen"
)

var segments = []string{
	"sports", "music", "tech", "travel", "food", "finance", "gaming", "news",
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn, err := grpc.NewClient(cfg.USSGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial %s: %v", cfg.USSGRPCAddr, err)
	}
	defer conn.Close()
	client := gen.NewMembershipServiceClient(conn)

	jobs := make(chan int, cfg.USSWorkers*2)
	var okCount, errCount atomic.Int64
	var wg sync.WaitGroup

	for w := 0; w < cfg.USSWorkers; w++ {
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
	for i := 0; i < cfg.USSEvents; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	elapsed := time.Since(start)
	log.Printf("done: ok=%d err=%d workers=%d events=%d addr=%s in %s (%.0f req/s)",
		okCount.Load(), errCount.Load(), cfg.USSWorkers, cfg.USSEvents, cfg.USSGRPCAddr,
		elapsed.Round(time.Millisecond),
		float64(okCount.Load())/elapsed.Seconds(),
	)
}
