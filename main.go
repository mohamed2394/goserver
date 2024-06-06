package main

import (
	"log"
	"net/http"
)

func main() {
	const port = "8080"

	mux := http.NewServeMux()
	setupRoutes(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	srv.ListenAndServe()
}

func setupRoutes(mux *http.ServeMux) {
	const filepathRoot = "."
	apiCfg := &apiConfig{
		fileserverHits: 0,
	}

	handler := http.FileServer(http.Dir(filepathRoot))
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(handler)))

	mux.Handle("/api/healthz", &readinessHandler{})
	mux.HandleFunc("/admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("/api/reset", apiCfg.resetHandler)
	mux.HandleFunc("/api/validate_chirp", ValidateChirpMiddleware(ValidateChirpHandler))
}
