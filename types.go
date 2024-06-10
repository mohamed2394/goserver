package main

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type ChirpRequest struct {
	Body string `json:"body"`
}
