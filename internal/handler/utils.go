package handler

import (
	"net/http"
	"slices"
	"strings"
)

func allowMethod(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	if slices.Contains(allowed, r.Method) {
		return true
	}

	w.Header().Set("Allow", strings.Join(allowed, ", "))
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

	return false
}
