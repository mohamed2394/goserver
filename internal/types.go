package internal

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type ChirpRequest struct {
	Body string `json:"body"`
}

type User struct {
	Id    int    `json:"id"`
	Email string `json:"email"`
}

type UserRequest struct {
	Email string `json:"email"`
}
