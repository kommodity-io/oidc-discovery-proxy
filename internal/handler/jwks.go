package handler

import "net/http"

const (
	JWKSPath = "/openid/v1/jwks"
)

func (h *OIDCDiscoveryProxyHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet, http.MethodHead, http.MethodOptions) {
		return
	}

	data, statusCode, err := h.handle(r.Context(), JWKSPath)
	if err != nil {
		http.Error(w, err.Error(), statusCode)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, err = w.Write(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}
