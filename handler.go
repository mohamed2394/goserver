package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Define the struct to hold state
type apiConfig struct {
	mu             sync.Mutex
	fileserverHits int
}

type readinessHandler struct{}

type ErrorResponse struct {
	Error string `json:"error"`
}

func ValidateChirpMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqBody ChirpRequest
		profaneWords := []string{"kerfuffle", "sharbert", "fornax"}

		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}

		if len(reqBody.Body) > 140 {
			respondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}

		cleanedBody := replaceProfaneWords(reqBody.Body, profaneWords)

		respondWithJSON(w, http.StatusOK, map[string]string{"cleaned_body": cleanedBody})

	}
}

// Function to replace profane words
func replaceProfaneWords(text string, profaneWords []string) string {
	words := strings.Fields(text) // Split the text into words
	replacement := "****"

	for i, word := range words {
		// Normalize the word to lowercase for comparison
		normalizedWord := strings.ToLower(word)

		// Check if the normalized word (stripped of punctuation) is in the profane words list
		for _, profaneWord := range profaneWords {
			if normalizedWord == profaneWord {
				words[i] = replacement // Replace the profane word
			}
		}
	}

	// Reconstruct the text from words
	return strings.Join(words, " ")
}

// handler for /api/validate_chirp endpoint
func ValidateChirpHandler(w http.ResponseWriter, r *http.Request) {
	// Respond with success message if validation passed

}

// middleware to increment the fileserverHits counter
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
