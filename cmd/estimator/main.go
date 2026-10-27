package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"github.com/selseleha/go-challenge/internal/segment"
	segmentgrpc "github.com/selseleha/go-challenge/internal/segment/adapter/grpc"
	segmenthttp "github.com/selseleha/go-challenge/internal/segment/adapter/http"
	"github.com/selseleha/go-challenge/internal/segment/adapter/memory"
	segmentredis "github.com/selseleha/go-challenge/internal/segment/adapter/redis"
)

func main() {
	repo := newRepository()
	svc := segment.NewService(repo, segment.SystemClock{})

	httpAddr := envOr("HTTP_ADDR", ":8080")
	grpcAddr := envOr("GRPC_ADDR", ":9090")

	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           segmenthttp.NewHandler(svc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcServer := grpc.NewServer()
	segmentgrpc.NewServer(svc).Register(grpcServer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)

	go func() {
		log.Printf("http estimate API listening on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			errCh <- err
			return
		}
		log.Printf("grpc ingest API listening on %s", grpcAddr)
		errCh <- grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		log.Print("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		grpcServer.GracefulStop()
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	}
}

func newRepository() segment.Repository {
	store := envOr("STORE", "memory")
	switch store {
	case "memory":
		log.Print("store: memory")
		return memory.NewRepository()
	case "redis":
		addr := envOr("REDIS_ADDR", "localhost:6379")
		client := goredis.NewClient(&goredis.Options{Addr: addr})
		if err := client.Ping(context.Background()).Err(); err != nil {
			log.Fatalf("redis ping %s: %v", addr, err)
		}
		log.Printf("store: redis (%s)", addr)
		return segmentredis.NewRepository(client)
	default:
		log.Fatalf("unknown STORE %q (want memory or redis)", store)
		return nil
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
