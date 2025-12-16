package handler

import (
	"net/http"

	"go.uber.org/zap"
)

const (
	// JWKSPath is the path for the JSON Web Key Set endpoint.
	JWKSPath = "/openid/v1/jwks"
)

// JWKS handles requests to the JSON Web Key Set endpoint.
func (h *OIDCDiscoveryProxyHandler) JWKS(writer http.ResponseWriter, request *http.Request) {
	if !allowMethod(writer, request, http.MethodGet, http.MethodHead, http.MethodOptions) {
		return
	}

	h.logger.Info("Handling JWKS request", zap.String("path", JWKSPath))

	data, statusCode, err := h.handle(request.Context(), JWKSPath)
	if err != nil {
		http.Error(writer, err.Error(), statusCode)

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)

	_, err = writer.Write(data)
	if err != nil {
		h.logger.Error("Failed to write JWKS response", zap.Error(err))

		return
	}
}
