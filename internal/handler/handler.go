// Package handler provides HTTP handlers for OIDC discovery proxy.
package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	// envCacheTTLSeconds is the environment variable used to configure the response cache TTL.
	envCacheTTLSeconds = "CACHE_TTL_SECONDS"
	// defaultCacheTTL is the default TTL for cached responses when envCacheTTLSeconds is unset.
	defaultCacheTTL = 60 * time.Second

	// kubernetesClientQPS is the sustained requests-per-second limit for the Kubernetes client.
	kubernetesClientQPS = 100
	// kubernetesClientBurst is the burst limit for the Kubernetes client.
	kubernetesClientBurst = 100
)

// OIDCDiscoveryProxyHandler handles OIDC discovery proxy requests.
type OIDCDiscoveryProxyHandler struct {
	client *kubernetes.Clientset
	logger *slog.Logger
	cache  *responseCache
}

// NewOIDCDiscoveryProxyHandler creates a new instance of OIDCDiscoveryProxyHandler.
func NewOIDCDiscoveryProxyHandler(logger *slog.Logger) (*OIDCDiscoveryProxyHandler, error) {
	client, err := createKubernetesClient()
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}

	ttl := getCacheTTL(logger)

	logger.Info("Response cache configured", "ttl", ttl)

	return &OIDCDiscoveryProxyHandler{
		client: client,
		logger: logger,
		cache:  newResponseCache(ttl),
	}, nil
}

//nolint:wrapcheck // Errors are handled in the calling functions.
func (h *OIDCDiscoveryProxyHandler) handle(ctx context.Context, path string) ([]byte, int, error) {
	if data, found := h.cache.get(path); found {
		return data, http.StatusOK, nil
	}

	bytes, err := h.client.RESTClient().Get().AbsPath(path).DoRaw(ctx)
	if err != nil {
		var kErr *kerrors.StatusError

		success := errors.As(err, &kErr)
		if !success {
			return nil, http.StatusInternalServerError, err
		}

		return nil, int(kErr.ErrStatus.Code), err
	}

	h.cache.set(path, bytes)

	return bytes, http.StatusOK, nil
}

// getCacheTTL resolves the response cache TTL from the environment, falling back to defaultCacheTTL.
func getCacheTTL(logger *slog.Logger) time.Duration {
	value := os.Getenv(envCacheTTLSeconds)
	if value == "" {
		return defaultCacheTTL
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		logger.Warn("Invalid "+envCacheTTLSeconds+" value, using default",
			"value", value, "default", defaultCacheTTL)

		return defaultCacheTTL
	}

	return time.Duration(seconds) * time.Second
}

func createKubernetesClient() (*kubernetes.Clientset, error) {
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		return createOutOfClusterKubernetesClient()
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster kubeconfig: %w", err)
	}

	applyClientRateLimits(config)

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return client, nil
}

func createOutOfClusterKubernetesClient() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homedir.HomeDir()
		if home == "" {
			return nil, ErrFailedToResolveKubeconfig
		}

		kubeconfig = path.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load local kubeconfig: %w", err)
	}

	applyClientRateLimits(config)

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return client, nil
}

// applyClientRateLimits configures the Kubernetes client's sustained and burst request rate limits.
func applyClientRateLimits(config *rest.Config) {
	config.QPS = kubernetesClientQPS
	config.Burst = kubernetesClientBurst
}
