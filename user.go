package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (cfg *apiConfig) addUser(w http.ResponseWriter, req *http.Request) {
	type userMail struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	user := userMail{}
	err := decoder.Decode(&user)
	if err != nil {
		log.Printf("Error decoding response body: %v", err)
		return
	}

	newUser, err := cfg.dbQueries.CreateUser(req.Context(), user.Email)
	if err != nil {
		log.Printf("Error creating new user: %v", err)
		return
	}

	type NewUser struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
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
