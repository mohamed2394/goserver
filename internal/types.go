package internal

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type ChirpRequest struct {
	Body string `json:"body"`
}

type User struct {
	Id       int    `json:"id"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type UserRequest struct {
	Password         string `json:"password"`
	Email            string `json:"email"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}
