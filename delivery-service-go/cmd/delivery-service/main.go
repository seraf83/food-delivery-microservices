// Command delivery-service runs either the HTTP API (`serve`) or the Kafka
// worker that assigns couriers to new orders (`consume`).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seraf83/food-delivery/delivery-service/internal/config"
	"github.com/seraf83/food-delivery/delivery-service/internal/events"
	"github.com/seraf83/food-delivery/delivery-service/internal/httpapi"
	"github.com/seraf83/food-delivery/delivery-service/internal/store"
	"github.com/seraf83/food-delivery/delivery-service/internal/worker"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: delivery-service <serve|consume>")
		os.Exit(2)
	}

	if err := run(os.Args[1], log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(mode string, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	st, err := store.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		return err
	}

	pub := events.NewPublisher(cfg.KafkaBrokers, cfg.DeliveriesTopic)
	defer pub.Close()

	switch mode {
	case "serve":
		return serve(ctx, cfg, httpapi.New(st, pub, log), log)
	case "consume":
		return worker.New(cfg, st, pub, log).Run(ctx)
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
}

func serve(ctx context.Context, cfg config.Config, h http.Handler, log *slog.Logger) error {
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: h, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Info("http server listening", "addr", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
