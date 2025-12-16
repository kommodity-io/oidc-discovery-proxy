package handler

import (
	"net/http"

	"go.uber.org/zap"
)

const (
	// OpenIDConfigPath is the path for the OpenID Connect configuration endpoint.
	OpenIDConfigPath = "/.well-known/openid-configuration"
)

// OpenIDConfig handles requests to the OpenID Connect configuration endpoint.
func (h *OIDCDiscoveryProxyHandler) OpenIDConfig(writer http.ResponseWriter, request *http.Request) {
	if !allowMethod(writer, request, http.MethodGet, http.MethodHead, http.MethodOptions) {
		return
	}

	h.logger.Info("Handling OpenID Configuration request", zap.String("path", OpenIDConfigPath))

	data, statusCode, err := h.handle(request.Context(), OpenIDConfigPath)
	if err != nil {
		http.Error(writer, err.Error(), statusCode)

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)

	_, err = writer.Write(data)
	if err != nil {
		h.logger.Error("Failed to write OpenID Configuration response", zap.Error(err))

		return
	}
}
