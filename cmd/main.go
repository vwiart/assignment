package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/vwiart/assignment/pkg/activity"
	"github.com/vwiart/assignment/storage"
	store "github.com/vwiart/assignment/storage/activity"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := startService(ctx); err != nil {
		logger.Error("failed to start service", "err", err)
		panic(err)
	}
}

func startService(ctx context.Context) error {
	logger := slog.Default()
	db, err := storage.Open(ctx, "data/db")
	if err != nil {
		return err
	}
	defer db.Close()

	if err := storage.ApplyMigrations(ctx, db); err != nil {
		return err
	}

	activityStore := store.NewActivity(db)
	a := activity.New(activityStore)

	mux := chi.NewMux()
	mux.Get("/activities/total", a.Aggregate())
	mux.Get("/users", a.ListHandler())
	mux.Post("/users/{user_id}", a.SubmitHandler())
	mux.Get("/users/{user_id}/total", a.AggregateByUser())
	mux.Delete("/users/{activity_id}", a.DeleteHandler())

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("server started", "port", port)
	return http.ListenAndServe(fmt.Sprintf(":%s", port), mux)
}
