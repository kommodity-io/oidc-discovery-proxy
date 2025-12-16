package main

import (
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/kommodity-io/oidc-discovery-proxy/internal/handler"
)

const (
	timeout     = 1 * time.Second
	defaultPort = "8080"
)

func main() {
	var ready atomic.Bool

	oidcHandler, err := handler.NewOIDCDiscoveryProxyHandler()
	if err != nil {
		panic(err)
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

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	// Mark the application as ready after successful initialization
	ready.Store(true)

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
