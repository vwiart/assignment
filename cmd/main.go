package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vwiart/assignment/pkg/activity"
	"github.com/vwiart/assignment/pkg/ui"
	"github.com/vwiart/assignment/storage"
	store "github.com/vwiart/assignment/storage/activity"
)

// shutdownTimeout is how long in-flight requests have to finish after SIGINT or SIGTERM.
const shutdownTimeout = 10 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := startService(ctx); err != nil {
		logger.Error("failed to start service", "err", err)
		os.Exit(1)
	}
}

func startService(ctx context.Context) error {
	logger := slog.Default()
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "data/db"
	}
	db, err := storage.Open(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := storage.ApplyMigrations(ctx, db); err != nil {
		return err
	}

	routes := chi.NewMux()
	routes.Get("/", ui.Handler())
	routes.Mount("/", activity.New(store.NewActivity(db)).Routes())

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{Addr: fmt.Sprintf(":%s", port), Handler: routes}
	errs := make(chan error, 1)
	go func() { errs <- server.ListenAndServe() }()
	logger.Info("server started", "port", port, "database", path)

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return server.Shutdown(ctx)
}
