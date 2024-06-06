package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Define the struct to hold state
type apiConfig struct {
	mu             sync.Mutex
	fileserverHits int
}

type readinessHandler struct{}

type ChirpRequest struct {
	Body string `json:"body"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}

func ValidateChirpMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode JSON request body
		var reqBody ChirpRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			errorResponse := ErrorResponse{Error: "Something went wrong"}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		// Check chirp length
		if len(reqBody.Body) > 140 {
			errorResponse := ErrorResponse{Error: "Chirp is too long"}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		// Call the next handler if validation passes
		next.ServeHTTP(w, r)
	}
}

// ValidateChirpHandler is the actual handler for /api/validate_chirp endpoint
func ValidateChirpHandler(w http.ResponseWriter, r *http.Request) {
	// Respond with success message if validation passed
	response := map[string]bool{"valid": true}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Middleware to increment the fileserverHits counter
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.mu.Lock()
		cfg.fileserverHits++
		cfg.mu.Unlock()
		next.ServeHTTP(w, r) // Call the next handler
	})
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	prevCount := cfg.fileserverHits
	cfg.mu.Lock()
	cfg.fileserverHits = 0
	cfg.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hits count was %d, ,now reset to 0", prevCount)))
}

// Handler to return the current hit count
func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits)))
}

// Handler to return OK for readiness
func (rh *readinessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
