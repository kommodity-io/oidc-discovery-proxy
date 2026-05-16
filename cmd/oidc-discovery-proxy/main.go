// Package main implements the OIDC Discovery Proxy server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kommodity-io/oidc-discovery-proxy/internal/handler"
	"github.com/lmittmann/tint"
)

const (
	timeout         = 1 * time.Second
	shutdownTimeout = 5 * time.Second
	defaultPort     = "8080"
)

var (
	//nolint:gochecknoglobals
	ready atomic.Bool
)

func main() {
	triggers := []os.Signal{
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, triggers...)

	logger := getLogger()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	mux, err := getOIDCMux(logger)
	if err != nil {
		logger.Error("Failed to create HTTP mux", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", server.Addr)
	if err != nil {
		logger.Error("Failed to bind listener", "error", err)
		os.Exit(1)
	}

	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	ready.Store(true)

	logger.Info("OIDC Discovery Proxy started successfully")

	sig := <-signals

	logger.Info("Received signal", "signal", sig.String())

	ready.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		logger.Error("Failed to shut down HTTP server", "error", err)
	}
}

func getOIDCMux(logger *slog.Logger) (*http.ServeMux, error) {
	oidcHandler, err := handler.NewOIDCDiscoveryProxyHandler(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC handler: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(handler.OpenIDConfigPath, oidcHandler.OpenIDConfig)
	mux.HandleFunc(handler.JWKSPath, oidcHandler.JWKS)
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(writer http.ResponseWriter, _ *http.Request) {
		if ready.Load() {
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("ready"))
		} else {
			writer.WriteHeader(http.StatusServiceUnavailable)
			_, _ = writer.Write([]byte("not ready"))
		}
	})

	return mux, nil
}

func getLogger() *slog.Logger {
	level := parseLogLevel(os.Getenv("LOG_LEVEL"))

	format := strings.ToLower(os.Getenv("LOG_FORMAT"))
	if format == "" {
		format = "json"
	}

	var handler slog.Handler

	switch format {
	case "console":
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: time.RFC3339,
		})
	default:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	return slog.New(handler)
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}
