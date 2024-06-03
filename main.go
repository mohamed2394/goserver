package main

import (
	"net/http"
)

const port = "8080"

func main() {

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(".")))

	srv := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	srv.ListenAndServe()
}
