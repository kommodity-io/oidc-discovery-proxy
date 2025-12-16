// Package main implements the OIDC Discovery Proxy server.
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kommodity-io/oidc-discovery-proxy/internal/handler"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	timeout     = 1 * time.Second
	defaultPort = "8080"
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

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, _ := config.Build()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	mux, err := getOIDCMux(logger)
	if err != nil {
		logger.Fatal("Failed to create HTTP mux", zap.Error(err))
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	// Mark the application as ready after successful initialization
	ready.Store(true)

	logger.Info("OIDC Discovery Proxy started successfully")

	sig := <-signals

	logger.Info("Received signal", zap.String("signal", sig.String()))
}

func getOIDCMux(logger *zap.Logger) (*http.ServeMux, error) {
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
