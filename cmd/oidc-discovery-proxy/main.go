package main

import (
	"net/http"
	"os"
	"time"

	"github.com/kommodity-io/oidc-discovery-proxy/internal/handler"
)

const (
	timeout     = 1 * time.Second
	defaultPort = "8080"
)

func main() {
	oidcHandler, err := handler.NewOIDCDiscoveryProxyHandler()
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(handler.OpenIDConfigPath, oidcHandler.OpenIDConfig)
	mux.HandleFunc(handler.JWKSPath, oidcHandler.JWKS)

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

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
