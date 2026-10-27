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

	"github.com/selseleha/go-challenge/internal/config"
	segmentgrpc "github.com/selseleha/go-challenge/internal/segment/adapter/grpc"
	segmenthttp "github.com/selseleha/go-challenge/internal/segment/adapter/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := wireApp(ctx, cfg)
	if err != nil {
		log.Fatalf("wire: %v", err)
	}
	defer app.Cleanup()

	httpServer := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: segmenthttp.NewHandler(segmenthttp.Deps{
			Estimator:  app.Service,
			Health:     app.Service,
			Resetter:   app.Service,
			AdminToken: cfg.AdminToken,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcServer := grpc.NewServer()
	segmentgrpc.NewServer(app.Service).Register(grpcServer)

	errCh := make(chan error, 2)

	go func() {
		log.Printf("http listening on %s (ttl=%s store=%s)", cfg.HTTPAddr, cfg.MembershipTTL, cfg.Store)
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
		log.Printf("grpc ingest listening on %s", cfg.GRPCAddr)
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
