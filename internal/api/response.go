package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// Response represents a standard API response envelope.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// writeJSON serializes data as JSON with the given HTTP status code.
func writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response with the given status code.
func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]interface{}{
		"success": false,
		"error":   msg,
	})
}

// writeSuccess writes a JSON success response with HTTP 200 OK and data envelope.
func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// getIDParam extracts the "id" route variable using gorilla/mux, falling back to query or URL path.
func getIDParam(r *http.Request) string {
	vars := mux.Vars(r)
	if id, ok := vars["id"]; ok && id != "" {
		return id
	}

	if id := r.URL.Query().Get("id"); id != "" {
		return id
	}

	path := strings.TrimRight(r.URL.Path, "/")
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[idx+1:]
	}

	return ""
}
