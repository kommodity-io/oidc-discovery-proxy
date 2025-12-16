package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type OIDCDiscoveryProxyHandler struct {
	client *kubernetes.Clientset
}

// NewOIDCDiscoveryProxyHandler creates a new instance of OIDCDiscoveryProxyHandler.
func NewOIDCDiscoveryProxyHandler() (*OIDCDiscoveryProxyHandler, error) {
	client, err := createKubernetesClient()
	if err != nil {
		return nil, fmt.Errorf("create in-cluster HTTP client: %w", err)
	}

	return &OIDCDiscoveryProxyHandler{
		client: client,
	}, nil
}

//nolint:wrapcheck // Errors are handled in the calling functions.
func (h *OIDCDiscoveryProxyHandler) handle(ctx context.Context, path string) ([]byte, int, error) {
	bytes, err := h.client.RESTClient().Get().AbsPath(path).DoRaw(ctx)
	if err != nil {
		var kErr *kerrors.StatusError

		success := errors.As(err, &kErr)
		if !success {
			return nil, http.StatusInternalServerError, err
		}

		return nil, int(kErr.ErrStatus.Code), err
	}

	return bytes, http.StatusOK, nil
}

func createKubernetesClient() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homedir.HomeDir()
		if home == "" {
			return nil, fmt.Errorf("failed to resolve kubeconfig: KUBECONFIG not set and home directory not found")
		}

		kubeconfig = path.Join(home, ".kube", "config")
	}

	isRunningOutsideCluster := os.Getenv("KUBERNETES_SERVICE_HOST") == ""
	if isRunningOutsideCluster {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load local kubeconfig: %w", err)
		}

		return kubernetes.NewForConfig(config)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return client, nil
}
