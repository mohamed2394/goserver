package database

import (
	// Other imports...
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	. "github.com/mohamed2394/goserver/internal"
	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	Path           string
	ChirpIdCounter int
	UserIdCounter  int
	Mux            *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	db := DB{
		Path:           path,
		ChirpIdCounter: 1,
		UserIdCounter:  1,
		Mux:            &sync.RWMutex{},
	}
	err := db.ensureDB()
	if err != nil {
		return nil, err
	}
	return &db, nil
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	log.Println("Creating a new chirp")

	chirp := Chirp{
		Id:   db.ChirpIdCounter,
		Body: body,
	}
	db.ChirpIdCounter++
	log.Printf("Assigned chirp ID: %d", chirp.Id)

	chirps, errC := db.GetChirps()
	if errC != nil {
		log.Println("Error getting chirps:", errC)
		return Chirp{}, errC
	}

	chirps = append(chirps, chirp)
	users, errH := db.GetUsers()
	if errH != nil {
		log.Println("Error getting Users:", errC)
		return Chirp{}, errC
	}

	errU := db.UpdateDB(users, chirps)
	if errU != nil {
		log.Println("Error writing database:", errU)
		return Chirp{}, errU
	}
	log.Println("Successfully created a new user")
	return chirp, nil
}

func (db *DB) CreateUser(email string, password string) (User, error) {
	log.Println("Creating a new user")

	// Check if email already exists
	users, err := db.GetUsers()
	if err != nil {
		log.Println("Error getting users:", err)
		return User{}, err
	}
	for _, user := range users {
		if user.Email == email {
			return User{}, errors.New("email already in use")
		}
	}

	// Hash the password
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Error hashing password:", err)
		return User{}, err
	}

	// Create new user
	user := User{
		Id:       db.UserIdCounter,
		Password: string(hashPassword),
		Email:    email,
	}
	db.UserIdCounter++
	log.Printf("Assigned user ID: %d", user.Id)

	// Append new user and update database
	users = append(users, user)
	chirps, err := db.GetChirps()
	if err != nil {
		log.Println("Error getting chirps:", err)
		return User{}, err
	}
	err = db.UpdateDB(users, chirps)
	if err != nil {
		log.Println("Error writing database:", err)
		return User{}, err
	}
	log.Println("Successfully created a new user")
	return user, nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	log.Println("Acquiring read lock for getting chirps")
	db.Mux.RLock()
	defer func() {
		log.Println("Releasing read lock after getting chirps")
		db.Mux.RUnlock()
	}()

	log.Println("Opening database file:", db.Path)
	file, err := os.Open(db.Path)
	if err != nil {
		log.Println("Error opening database file:", err)
		return nil, err
	}
	defer func() {
		log.Println("Closing database file")
		file.Close()
	}()

	log.Println("Checking if database file is empty")
	fileInfo, err := file.Stat()
	if err != nil {
		log.Println("Error stating database file:", err)
		return nil, err
	}
	if fileInfo.Size() == 0 {
		log.Println("Database file is empty")
		return []Chirp{}, nil
	}

	log.Println("Decoding database file")
	decoder := json.NewDecoder(file)
	var dbs DBStructure

	err = decoder.Decode(&dbs)
	if err != nil {
		log.Println("Error decoding database file:", err)
		return nil, err
	}

	log.Println("Collecting chirps from decoded data")
	var chirps []Chirp
	for _, chirp := range dbs.Chirps {
		chirps = append(chirps, chirp)
	}

	log.Println("Sorting chirps by ID")
	sort.Slice(chirps, func(i, j int) bool { return chirps[i].Id < chirps[j].Id })
	return chirps, nil
}

func (db *DB) GetUsers() ([]User, error) {
	log.Println("Acquiring read lock for getting chirps")
	db.Mux.RLock()
	defer func() {
		log.Println("Releasing read lock after getting chirps")
		db.Mux.RUnlock()
	}()

	log.Println("Opening database file:", db.Path)
	file, err := os.Open(db.Path)
	if err != nil {
		log.Println("Error opening database file:", err)
		return nil, err
	}
	defer func() {
		log.Println("Closing database file")
		file.Close()
	}()

	log.Println("Checking if database file is empty")
	fileInfo, err := file.Stat()
	if err != nil {
		log.Println("Error stating database file:", err)
		return nil, err
	}
	if fileInfo.Size() == 0 {
		log.Println("Database file is empty")
		return []User{}, nil
	}

	log.Println("Decoding database file")
	decoder := json.NewDecoder(file)
	var dbs DBStructure

	err = decoder.Decode(&dbs)
	if err != nil {
		log.Println("Error decoding database file:", err)
		return nil, err
	}

	log.Println("Collecting users from decoded data")
	var users []User
	for _, user := range dbs.Users {
		users = append(users, user)
	}

	log.Println("Sorting ysers by ID")
	sort.Slice(users, func(i, j int) bool { return users[i].Id < users[j].Id })
	return users, nil
}

func (db *DB) GetUser(email, password string) (User, error) {
	// Retrieve all users from the database
	users, errC := db.GetUsers()
	if errC != nil {
		log.Println("Error getting users:", errC)
		return User{}, errC
	}

	// Iterate through the users to find a match by email
	for _, u := range users {
		if u.Email == email {
			// Compare the provided password with the stored hashed password
			err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
			if err != nil {
				log.Println("Password does not match:", err)
				return User{}, errors.New("invalid email or password")
			}
			// Return the user if the password matches
			return u, nil
		}
	}

	// Handle case where no user was found
	log.Println("No user was found for this email")
	return User{}, errors.New("no user was found for this email")
}

func (db *DB) UpdateUser(id int, newEmail, newPassword, refresh_token string) error {

	users, errU := db.GetUsers()
	if errU != nil {
		return errU
	}
	chirps, errC := db.GetChirps()
	if errC != nil {
		return errC
	}

	userUpdated := false
	for i := range users {
		if users[i].Id == id {
			users[i].Email = newEmail

			// Hash new password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			users[i].Password = string(hashedPassword)
			users[i].RefreshToken = refresh_token
			refreshExpirationDate := time.Now().Add(60 * 24 * time.Hour) // 60 days from now
			users[i].RefreshExpirationDate = refreshExpirationDate
			userUpdated = true
			break
		}
	}

	if !userUpdated {
		return fmt.Errorf("user with ID %d not found", id)
	}

	// Save updated users and chirps back to the database
	db.UpdateDB(users, chirps)

	return nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	log.Println("Checking if database file exists")
	_, err := os.Stat(db.Path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Database file does not exist, creating it")
			file, err := os.Create(db.Path)
			if err != nil {
				log.Printf("Error creating file: %v", err)
				return fmt.Errorf("error creating file: %w", err)
			}
			file.Close()
			err = os.Chmod(db.Path, 0644)
			if err != nil {
				log.Printf("Error setting file permissions: %v", err)
				return fmt.Errorf("error setting file permissions: %w", err)
			}
			return nil
		} else {
			log.Printf("Error checking file: %v", err)
			return fmt.Errorf("error checking file: %w", err)
		}
	}
	log.Println("Database file exists")
	return nil
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {
	log.Println("Ensuring database file exists")
	err := db.ensureDB()
	if err != nil {
		log.Println("Error ensuring database file exists:", err)
		return err
	}

	log.Println("Marshaling database structure to JSON")
	data, errD := json.Marshal(dbStructure)
	if errD != nil {
		log.Println("Error marshaling database structure to JSON:", errD)
		return errD
	}

	log.Println("Acquiring write lock for writing to the database")
	db.Mux.Lock()
	defer func() {
		log.Println("Releasing write lock after writing to the database")
		db.Mux.Unlock()
	}()

	log.Println("Writing data to database file:", db.Path)
	errW := os.WriteFile(db.Path, data, 0644)
	if errW != nil {
		log.Println("Error writing data to database file:", errW)
		return errW
	}

	log.Println("Successfully wrote data to the database")
	return nil
}

func (db *DB) UpdateDB(users []User, chirps []Chirp) error {
	dbs := DBStructure{
		Users:  make(map[int]User),
		Chirps: make(map[int]Chirp),
	}
	for _, u := range users {
		dbs.Users[u.Id] = u
	}
	for _, c := range chirps {
		dbs.Chirps[c.Id] = c
	}
	return db.writeDB(dbs)
}
