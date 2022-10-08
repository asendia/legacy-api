package api

import (
	"errors"
	"net/http"
)

// CORSEnabledFunction is an example of setting CORS headers.
// For more information about CORS and CORS preflight requests, see
// https://developer.mozilla.org/en-US/docs/Glossary/Preflight_request.
func VerifyCORS(w http.ResponseWriter, r *http.Request) (httpResponseCode int, err error) {
	allowedOrigins := []string{"https://sejiwo.com", "http://localhost:4173", "http://localhost:5173"}
	allowedOrigin := ""
	for _, origin := range allowedOrigins {
		if origin == r.Header.Get("origin") {
			allowedOrigin = origin
		}
	}
	if allowedOrigin == "" {
		http.Error(w, "CORS error", http.StatusForbidden)
		return http.StatusForbidden, errors.New("Origin doesn't exist in allowedOrigins")
	}
	// Set CORS headers for the preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return http.StatusNoContent, nil
	}
	// Set CORS headers for the main request.
	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	return http.StatusAccepted, nil
}
