package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jpabloVega/Chirpy/internal/auth"
	"github.com/jpabloVega/Chirpy/internal/database"
)

func (cfg *apiConfig) addUser(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	user := userMail{}
	err := decoder.Decode(&user)
	if err != nil {
		log.Printf("Error decoding response body: %v", err)
		return
	}

	hashedPass, err := auth.HashPassword(user.Password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		respondWithError(w, 400, "Error hashing user password")
		return
	}

	userParams := database.CreateUserParams{
		Email:          user.Email,
		HashedPassword: hashedPass,
	}

	newUser, err := cfg.dbQueries.CreateUser(req.Context(), userParams)
	if err != nil {
		log.Printf("Error creating new user: %v", err)
		return
	}

	fixedJsonUser := NewUser{
		Id:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}

	byteUser, err := json.Marshal(fixedJsonUser)
	if err != nil {
		log.Printf("Error marshalling user: %v", err)
		return
	}

	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")
	w.Write(byteUser)

}

func (cfg *apiConfig) logUser(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	user := userMail{}
	err := decoder.Decode(&user)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		respondWithError(w, 400, "Error decoding request")
		return
	}

	dbUser, err := cfg.dbQueries.GetUserByEmail(req.Context(), user.Email)
	if err != nil {
		log.Printf("Error getting user by email: %v", err)
		respondWithError(w, 400, "Error searching user by email")
		return
	}

	is_auth, err := auth.CheckPasswordHash(user.Password, dbUser.HashedPassword)
	if err != nil {
		log.Printf("Error checking password: %v", err)
		respondWithError(w, 400, "Error checking password")
		return
	}

	if is_auth {
		userData := NewUser{
			Id:        dbUser.ID,
			CreatedAt: dbUser.CreatedAt,
			UpdatedAt: dbUser.UpdatedAt,
			Email:     dbUser.Email,
		}
		respondWithJSON(w, 200, userData)
	} else {
		respondWithError(w, 401, "Incorrect email or password")
	}
}

type NewUser struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type userMail struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
