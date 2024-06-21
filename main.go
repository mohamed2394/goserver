package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	e "github.com/mohamed2394/goserver/internal"
	d "github.com/mohamed2394/goserver/internal/database"
)

func main() {
	const port = "8080"

	// Set up database
	dbPath := "internal/database/database.json"
	db, err := d.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to set up database: %v\n", err)
	}
	defer os.Remove(dbPath) // Ensure the database file is deleted on exit

	// by default, godotenv will look for a file named .env in the current directory
	// Set up server and routes
	mux := http.NewServeMux()
	setupRoutes(mux, db)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)

	// Handle graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := srv.Close(); err != nil {
		log.Fatalf("Server Close: %v\n", err)
	}
}
func setupRoutes(mux *http.ServeMux, db *d.DB) {
	errV := godotenv.Load()
	if errV != nil {
		log.Fatal("Error loading .env file")
	}

	secretKey := os.Getenv("JWT_SECRET")

	const filepathRoot = "."
	apiCfg := &apiConfig{
		fileserverHits: 0,
		secretKey:      secretKey,
	}

	chirpH := chirpHandler{
		db:     db,
		apiCfg: apiCfg,
	}

	userH := userHandler{
		db:     db,
		apiCfg: apiCfg,
	}

	handler := http.FileServer(http.Dir(filepathRoot))
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(handler)))

	mux.Handle("/api/healthz", &readinessHandler{})
	mux.HandleFunc("/admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("/api/reset", apiCfg.resetHandler)
	mux.HandleFunc("POST /api/users", userH.createUserHandler)
	mux.HandleFunc("POST /api/login", userH.loginUserHandler)
	mux.HandleFunc("PUT /api/users", userH.updateUserHandler)
	mux.HandleFunc("POST /api/refresh", userH.refreshToken)
	mux.HandleFunc("POST /api/revoke", userH.revokeToken)

	mux.HandleFunc("GET /api/chirps/{CHIRPID}", chirpH.getChirpByIdHandler)

	mux.HandleFunc("DELETE /api/chirps/{CHIRPID}", chirpH.deleteChirpHandler)

	mux.HandleFunc("/api/chirps", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			chirpH.postChirpsHandler(w, r)
		case http.MethodGet:
			chirpH.getChirpsHandler(w, r)
		default:
			e.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})
}
