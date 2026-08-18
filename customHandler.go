package main

import (
	"Chirpy/internal/database"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"unicode/utf8"

	"Chirpy/internal/database"

	"github.com/google/uuid"
)

func handlerReadiness(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, req *http.Request) {

	type chirpBody struct {
		Body   string    `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(req.Body)
	chirp := chirpBody{}
	err := decoder.Decode(&chirp)
	if err != nil {
		log.Printf("Error decoding parameters: %v", err)
		return
	}

	if utf8.RuneCountInString(chirp.Body) > 140 {
		respondWithError(w, 400, "Chirps over 140 characters, are not allowed")
		return
	}

	filteredChirp := removeProfanity(chirp.Body)

	chirpParams := database.CreateChirpParams{
		Body:   filteredChirp,
		UserID: chirp.UserId,
	}

	dbRes, err := cfg.dbQueries.CreateChirp(req.Context(), chirpParams)
	if err != nil {
		respondWithError(w, 400, "Error adding chirp to database")
	}

	type ChirpResponse struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}

	chirpRes := ChirpResponse{
		Id:        dbRes.ID,
		CreatedAt: dbRes.CreatedAt,
		UpdatedAt: dbRes.UpdatedAt,
		Body:      dbRes.Body,
		UserId:    dbRes.UserID,
	}
	chirpJson, err := json.Marshal(&chirpRes)
	if err != nil {
		respondWithError(w, 400, "Error marshalling chirp")
	}

	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")
	w.Write(chirpJson)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error marshalling: %v", err))
	}
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
}
