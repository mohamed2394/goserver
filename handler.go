package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	. "github.com/mohamed2394/goserver/internal"
	. "github.com/mohamed2394/goserver/internal/database"
)

type apiConfig struct {
	mu             sync.Mutex
	fileserverHits int
	secretKey      string
}

type readinessHandler struct{}
type userHandler struct {
	db     *DB
	apiCfg *apiConfig
}

type chirpHandler struct {
	db     *DB
	apiCfg *apiConfig
}

func (ch *chirpHandler) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirps, err := ch.db.GetChirps()
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Something went wrong")
		return
	}

	RespondWithJSON(w, http.StatusOK, chirps)
}

func (ch *chirpHandler) getChirpByIdHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the chirp ID from the URL
	stringId := r.PathValue("CHIRPID")
	id, err := strconv.Atoi(stringId)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	// Convert to zero-based index
	zeroBasedID := id - 1

	// Fetch all chirps from the database
	chirps, err := ch.db.GetChirps()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to load chirps from the database")
		return
	}

	// Ensure the zero-based chirp ID is within the valid range
	if zeroBasedID < 0 || zeroBasedID >= len(chirps) {
		RespondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	// Fetch the chirp
	chirp := chirps[zeroBasedID]

	// Respond with the chirp in JSON format
	RespondWithJSON(w, http.StatusOK, chirp)
}

func (ch *chirpHandler) postChirpsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a POST request on /api/chirps")

	var reqBody ChirpRequest
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Printf("Error decoding JSON: %v", err)
		RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		RespondWithError(w, http.StatusUnauthorized, "Authorization header missing or malformed")
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(ch.apiCfg.secretKey), nil
	})
	if err != nil || !token.Valid {
		RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	// Extract claims
	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		RespondWithError(w, http.StatusUnauthorized, "Invalid token claims")
		return
	}
	userId, err := strconv.Atoi((*claims)["sub"].(string))
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid user ID in token")
		return
	}

	if len(reqBody.Body) > 140 {
		log.Println("Chirp is too long")
		RespondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanedBody := replaceProfaneWords(reqBody.Body, profaneWords)
	log.Printf("Cleaned chirp body: %s", cleanedBody)

	chirp, err := ch.db.CreateChirp(cleanedBody)
	if err != nil {
		log.Printf("Failed to save chirp: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to save chirp")
		return
	}

	chirp.AuthorId = userId
	log.Printf("Chirp created with ID: %d", chirp.Id)
	RespondWithJSON(w, http.StatusCreated, chirp)
}

func (ch *chirpHandler) deleteChirpHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a delete request on /api/chirps/{chirpID}")

	// Extract the JWT from the Authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		RespondWithError(w, http.StatusUnauthorized, "Authorization header missing or malformed")
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(ch.apiCfg.secretKey), nil
	})
	if err != nil || !token.Valid {
		RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	// Extract claims
	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		RespondWithError(w, http.StatusUnauthorized, "Invalid token claims")
		return
	}
	userId, err := strconv.Atoi((*claims)["sub"].(string))
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid user ID in token")
		return
	}

	// Extract chirpID from URL
	chirpID := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	id, err := strconv.Atoi(chirpID)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	// Fetch the chirp from the database
	chirp, err := ch.db.GetChirp(id)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	// Authorization check: Ensure the user is the author of the chirp
	if chirp.AuthorId != userId {
		RespondWithError(w, http.StatusForbidden, "You are not authorized to delete this chirp")
		return
	}

	// Delete the chirp
	err = ch.db.DeleteChirp(id)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete chirp")
		return
	}

	// Respond with 204 No Content
	w.WriteHeader(http.StatusNoContent)
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

func (uh *userHandler) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody UserRequest
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Printf("Error decoding JSON: %v", err)
		RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Create user
	user, err := uh.db.CreateUser(reqBody.Email, reqBody.Password)
	if err != nil {
		if err.Error() == "email already in use" {
			RespondWithError(w, http.StatusConflict, "Email already in use")
		} else {
			log.Printf("Failed to create user: %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
		}
		return
	}

	log.Printf("User created with ID: %d", user.Id)

	// Respond with user info, excluding password
	response := struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
	}{
		Id:    user.Id,
		Email: user.Email,
	}
	RespondWithJSON(w, http.StatusCreated, response)
}

func (uh *userHandler) loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody UserRequest
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Printf("Error decoding JSON: %v", err)
		RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	user, errU := uh.db.GetUser(reqBody.Email, reqBody.Password)
	if errU != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}
	claims := &jwt.MapClaims{
		"iss": "chirpy",
		"iat": time.Now().UTC().Unix(),
		"exp": time.Now().UTC().Add(time.Hour).Unix(),
		"sub": strconv.Itoa(user.Id),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(uh.apiCfg.secretKey))
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Error generating token")
		return
	}

	refresh := make([]byte, 32)
	_, errR := rand.Read(refresh)

	if errR != nil {
		RespondWithError(w, http.StatusInternalServerError, "Error generating refresh token")
		return

	}
	refreshToken := hex.EncodeToString(refresh)

	uh.db.UpdateUser(user.Id, user.Email, user.Password, refreshToken)
	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":            user.Id,
		"email":         user.Email,
		"token":         tokenString,
		"refresh_token": refreshToken,
	})

}

func (uh *userHandler) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody UpdateUserRequest
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Extract the token from the request headers
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		RespondWithError(w, http.StatusUnauthorized, "Authorization header missing or malformed")
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(uh.apiCfg.secretKey), nil
	})
	if err != nil || !token.Valid {
		RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	// Extract claims
	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		RespondWithError(w, http.StatusUnauthorized, "Invalid token claims")
		return
	}
	userId, err := strconv.Atoi((*claims)["sub"].(string))
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid user ID in token")
		return
	}

	// Update user in the database
	err = uh.db.UpdateUser(userId, reqBody.Email, reqBody.Password, "")
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Respond with updated user info
	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":    userId,
		"email": reqBody.Email,
	})
}

func (uh *userHandler) refreshToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		RespondWithError(w, http.StatusUnauthorized, "Authorization header missing or malformed")
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	users, errU := uh.db.GetUsers()
	if errU != nil {
		RespondWithError(w, http.StatusUnauthorized, errU.Error())
		return

	}
	for _, user := range users {
		if user.RefreshToken == tokenString && time.Now().Before(user.RefreshExpirationDate) {
			claims := &jwt.MapClaims{
				"iss": "chirpy",
				"iat": time.Now().UTC().Unix(),
				"exp": time.Now().UTC().Add(time.Hour).Unix(),
				"sub": strconv.Itoa(user.Id),
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := token.SignedString([]byte(uh.apiCfg.secretKey))
			if err != nil {
				RespondWithError(w, http.StatusInternalServerError, "Error generating token")
				return
			}
			RespondWithJSON(w, http.StatusOK, map[string]interface{}{
				"token": tokenString,
			})
			break
		} else {
			RespondWithError(w, http.StatusUnauthorized, "Error generating token")
			return

		}
	}

}

func (uh *userHandler) revokeToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		RespondWithError(w, http.StatusUnauthorized, "Authorization header missing or malformed")
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	users, errU := uh.db.GetUsers()
	if errU != nil {
		RespondWithError(w, http.StatusUnauthorized, errU.Error())
		return
	}

	chirps, errC := uh.db.GetChirps()
	if errC != nil {
		RespondWithError(w, http.StatusInternalServerError, "Error fetching chirps")
		return
	}

	userFound := false
	for i := range users {
		if users[i].RefreshToken == tokenString {
			userFound = true
			users[i].RefreshToken = ""
			pastDate := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
			users[i].RefreshExpirationDate = pastDate
		}
	}

	if !userFound {
		RespondWithError(w, http.StatusUnauthorized, "NO user Found for this token")
		return
	}

	err := uh.db.UpdateDB(users, chirps)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Error updating database")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
