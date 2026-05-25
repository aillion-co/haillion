package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"haillion/services/gateway/internal/proxy"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	port := flag.String("port", "8080", "HTTP port to listen on")
	flag.Parse()

	// Initialize slog JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("starting API gateway", "port", *port)

	// Fetch target service URLs from environment or local defaults
	serviceURLs := map[string]string{
		"/api/identity":     getEnv("IDENTITY_URL", "http://localhost:8081"),
		"/api/matching":     getEnv("MATCHING_URL", "http://localhost:8082"),
		"/api/trip":         getEnv("TRIP_URL", "http://localhost:8083"),
		"/api/billing":      getEnv("BILLING_URL", "http://localhost:8084"),
		"/api/review":       getEnv("REVIEW_URL", "http://localhost:8085"),
		"/api/notification": getEnv("NOTIFICATION_URL", "http://localhost:8086"),
		"/":                 getEnv("FRONTEND_URL", "http://localhost:3000"),
	}

	for path, target := range serviceURLs {
		slog.Info("registering route mapping", "prefix", path, "target", target)
	}

	// Wire up layers
	mux, err := proxy.NewGatewayRouter(serviceURLs)
	if err != nil {
		slog.Error("failed to create gateway router", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              ":" + *port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	shutdownError := make(chan error)
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutting down API gateway gracefully...")

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

	slog.Info("API gateway stopped successfully")
}
