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

	"github.com/selseleha/go-challenge/internal/config"
	"github.com/selseleha/go-challenge/internal/segment"
	segmentgrpc "github.com/selseleha/go-challenge/internal/segment/adapter/grpc"
	segmenthttp "github.com/selseleha/go-challenge/internal/segment/adapter/http"
	"github.com/selseleha/go-challenge/internal/segment/adapter/memory"
	segmentredis "github.com/selseleha/go-challenge/internal/segment/adapter/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	repo := newRepository(cfg)
	svc := segment.NewService(repo, segment.SystemClock{})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           segmenthttp.NewHandler(svc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcServer := grpc.NewServer()
	segmentgrpc.NewServer(svc).Register(grpcServer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)

	go func() {
		log.Printf("http estimate API listening on %s (ttl=%s store=%s)", cfg.HTTPAddr, cfg.MembershipTTL, cfg.Store)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", cfg.GRPCAddr)
		if err != nil {
			errCh <- err
			return
		}
		log.Printf("grpc ingest API listening on %s", cfg.GRPCAddr)
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

func newRepository(cfg config.Config) segment.Repository {
	switch cfg.Store {
	case "memory":
		log.Print("store: memory")
		return memory.NewRepository(cfg.MembershipTTL)
	case "redis":
		client := goredis.NewClient(&goredis.Options{Addr: cfg.RedisAddr})
		if err := client.Ping(context.Background()).Err(); err != nil {
			log.Fatalf("redis ping %s: %v", cfg.RedisAddr, err)
		}
		log.Printf("store: redis (%s)", cfg.RedisAddr)
		return segmentredis.NewRepository(client, cfg.MembershipTTL)
	default:
		log.Fatalf("unknown STORE %q", cfg.Store)
		return nil
	}
}
