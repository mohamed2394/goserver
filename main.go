package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	e "github.com/mohamed2394/goserver/internal"
	d "github.com/mohamed2394/goserver/internal/database"
)

func main() {
	const port = "8080"

	// Set up database
	dbPath := "C:\\Users\\GEEK\\Desktop\\goserver\\internal\\database\\database.json"
	db, err := d.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to set up database: %v\n", err)
	}
	defer os.Remove(dbPath) // Ensure the database file is deleted on exit

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
	const filepathRoot = "."
	apiCfg := &apiConfig{
		fileserverHits: 0,
	}

	chirpH := chirpHandler{
		db: db,
	}

	userH := userHandler{
		db: db,
	}

	handler := http.FileServer(http.Dir(filepathRoot))
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(handler)))

	mux.Handle("/api/healthz", &readinessHandler{})
	mux.HandleFunc("/admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("/api/reset", apiCfg.resetHandler)
	mux.HandleFunc("POST /api/users", userH.postUserHandler)

	mux.HandleFunc("GET /api/chirps/{CHIRPID}", chirpH.getChirpByIdHandler)

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
