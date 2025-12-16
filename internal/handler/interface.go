package handler

import "net/http"

type OIDCDiscoveryProxy interface {
	OpenIDConfig(w http.ResponseWriter, r *http.Request)
	JWKS(w http.ResponseWriter, r *http.Request)
}
