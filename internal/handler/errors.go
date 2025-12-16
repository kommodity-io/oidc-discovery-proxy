//nolint:lll
package handler

import "errors"

var (
	// ErrFailedToResolveKubeconfig is returned when the kubeconfig cannot be resolved because KUBECONFIG is not set and the home directory is not found.
	ErrFailedToResolveKubeconfig = errors.New("failed to resolve kubeconfig: KUBECONFIG not set and home directory not found")
)
