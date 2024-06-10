package database

import (
	// Other imports...
	"sync"

	. "github.com/mohamed2394/goserver/internal"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}
