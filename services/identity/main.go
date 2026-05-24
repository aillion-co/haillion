package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aillion/services/identity/internal/api"
	"aillion/services/identity/internal/repository"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	port := flag.String("port", "8081", "HTTP port to listen on")
	dbURL := flag.String("db", "postgres://postgres:postgres@localhost:5432/aillion?sslmode=disable", "PostgreSQL database connection URL")
	flag.Parse()

	// Initialize slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("starting identity service", "port", *port)

	// Open database connection
	db, err := sql.Open("pgx", *dbURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}()

	// Verify database connection is alive
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		slog.Warn("database connection not ready, starting in mockable mode for local development", "error", err)
	} else {
		slog.Info("connected to database successfully")
	}

	// Wire up layers
	repo := repository.NewPostgresUserRepository(db)
	mux := api.NewServer(repo)

	server := &http.Server{
		Addr:    ":" + *port,
		Handler: mux,
	}

	// Setup graceful shutdown
	shutdownError := make(chan error)
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutting down identity service gracefully...")

		ctx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		shutdownError <- server.Shutdown(ctx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	if err := <-shutdownError; err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("identity service stopped successfully")
}
