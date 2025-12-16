package handler

import "net/http"

// OIDCDiscoveryProxy defines the interface for handling OIDC discovery endpoints.
type OIDCDiscoveryProxy interface {
	OpenIDConfig(w http.ResponseWriter, r *http.Request)
	JWKS(w http.ResponseWriter, r *http.Request)
}
