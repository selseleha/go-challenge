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

	"google.golang.org/grpc"

	"github.com/selseleha/go-challenge/internal/segment"
	segmentgrpc "github.com/selseleha/go-challenge/internal/segment/adapter/grpc"
	segmenthttp "github.com/selseleha/go-challenge/internal/segment/adapter/http"
	"github.com/selseleha/go-challenge/internal/segment/adapter/memory"
)

func main() {
	repo := memory.NewRepository()
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
