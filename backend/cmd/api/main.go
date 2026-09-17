// Command api serves the calculator HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PakiQuian/react-go-calc/backend/internal/api"
)

const (
	defaultPort = "8080"

	// http.ListenAndServe applies no timeouts at all, so a client that opens a
	// connection and stops sending holds a goroutine indefinitely. These are
	// generous for requests this small and still bound the damage.
	readTimeout  = 5 * time.Second
	writeTimeout = 10 * time.Second
	idleTimeout  = 60 * time.Second

	// shutdownTimeout bounds how long in-flight requests have to finish after
	// SIGTERM, which is what `docker compose down` sends.
	shutdownTimeout = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("server stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	server := &http.Server{
		Addr:         ":" + port(),
		Handler:      api.NewRouter(logger),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// Listen for termination before starting, so a signal arriving immediately
	// is not missed.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining connections")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		// Requests still running when the deadline passed are cut off here.
		return server.Close()
	}

	logger.Info("shutdown complete")
	return nil
}

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return defaultPort
}
