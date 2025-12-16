package handler

import (
	"net/http"
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

	data, statusCode, err := h.handle(request.Context(), OpenIDConfigPath)
	if err != nil {
		http.Error(writer, err.Error(), statusCode)

		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)

	_, err = writer.Write(data)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)

		return
	}
}
