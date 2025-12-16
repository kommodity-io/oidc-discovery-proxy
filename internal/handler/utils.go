package handler

import (
	"net/http"
	"slices"
	"strings"
)

func allowMethod(writer http.ResponseWriter, request *http.Request, allowed ...string) bool {
	if slices.Contains(allowed, request.Method) {
		return true
	}

	writer.Header().Set("Allow", strings.Join(allowed, ", "))
	http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)

	return false
}
