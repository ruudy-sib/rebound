package http

import (
	"encoding/json"
	"net/http"
)

// respondJSON writes a JSON response with the given status code and payload.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Encoding errors are not recoverable at this point, so we ignore the return.
	_ = json.NewEncoder(w).Encode(data)
}
