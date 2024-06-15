package internal

import "time"

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type ChirpRequest struct {
	Body string `json:"body"`
}

type User struct {
	Id                    int       `json:"id"`
	Password              string    `json:"password"`
	Email                 string    `json:"email"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshExpirationDate time.Time `json:"refresh_expiration_date"`
}

type UserRequest struct {
	Password         string `json:"password"`
	Email            string `json:"email"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type UpdateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
