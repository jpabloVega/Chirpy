package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jpabloVega/Chirpy/internal/auth"
	"github.com/jpabloVega/Chirpy/internal/database"
)

func handlerReadiness(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, req *http.Request) {

	userToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("Error getting the request header: %v", err)
		respondWithError(w, 400, "Invalid header")
		return
	}

	userID, err := auth.ValidateJWT(userToken, cfg.secret)
	if err != nil {
		log.Printf("Error validating the token: %v", err)
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type chirpBody struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	chirp := chirpBody{}
	err = decoder.Decode(&chirp)
	if err != nil {
		log.Printf("Error decoding parameters: %v", err)
		respondWithError(w, 400, "Error decoding request")
		return
	}

	if utf8.RuneCountInString(chirp.Body) > 140 {
		respondWithError(w, 400, "Chirps over 140 characters, are not allowed")
		return
	}

	filteredChirp := removeProfanity(chirp.Body)

	chirpParams := database.CreateChirpParams{
		Body:   filteredChirp,
		UserID: userID,
	}

	dbRes, err := cfg.dbQueries.CreateChirp(req.Context(), chirpParams)
	if err != nil {
		log.Printf("Database error: %v", err)
		respondWithError(w, 400, "Error adding chirp to database")
		return
	}

	chirpRes := ChirpResponse{
		Id:        dbRes.ID,
		CreatedAt: dbRes.CreatedAt,
		UpdatedAt: dbRes.UpdatedAt,
		Body:      dbRes.Body,
		UserId:    dbRes.UserID,
	}
	respondWithJSON(w, 201, chirpRes)
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.dbQueries.GetChirps(req.Context())
	if err != nil {
		log.Printf("Error getting chirps from db: %v", err)
	}

	var chirpArr []ChirpResponse

	for _, chirp := range chirps {
		chirpParams := ChirpResponse{
			Id:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserId:    chirp.UserID,
		}
		chirpArr = append(chirpArr, chirpParams)
	}

	dat, err := json.Marshal(chirpArr)
	if err != nil {
		log.Printf("Error marshalling getChirps: %v", err)
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, req *http.Request) {
	chirpId := req.PathValue("chirpID")
	parsedID, err := uuid.Parse(chirpId)
	if err != nil {
		respondWithError(w, 400, "Invalid id")
		return
	}

	foundChirp, err := cfg.dbQueries.GetChirp(req.Context(), parsedID)
	if err != nil {
		respondWithError(w, 404, "Chirp not found")
		return
	}

	chirpParams := ChirpResponse{
		Id:        foundChirp.ID,
		CreatedAt: foundChirp.CreatedAt,
		UpdatedAt: foundChirp.UpdatedAt,
		Body:      foundChirp.Body,
		UserId:    foundChirp.UserID,
	}

	respondWithJSON(w, 200, chirpParams)
}

func (cfg *apiConfig) refreshToken(w http.ResponseWriter, req *http.Request) {
	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "Invalid request header")
		return
	}

	currToken, err := cfg.dbQueries.GetRefreshToken(req.Context(), bearerToken)
	if err != nil {
		respondWithError(w, 401, "Token not found")
		return
	}

	if currToken.RevokedAt.Valid {
		respondWithError(w, 401, "Token access revoked")
		return
	}

	newJWT, err := auth.MakeJWT(currToken.UserID, cfg.secret, time.Second*60)
	if err != nil {
		log.Printf("Error creating authentication: %v", err)
		respondWithError(w, 400, "Error creating auth")
		return
	}

	type responseJWTToken struct {
		Token string `json:"token"`
	}
	responseToken := responseJWTToken{
		Token: newJWT,
	}
	respondWithJSON(w, 200, responseToken)

}

func (cfg *apiConfig) revokeToken(w http.ResponseWriter, req *http.Request) {
	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "Invalid request header")
		return
	}

	err = cfg.dbQueries.RevokeToken(req.Context(), bearerToken)
	if err != nil {
		respondWithError(w, 401, "Token not found")
		return
	}

	w.WriteHeader(204)

}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error marshalling: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

type ChirpResponse struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
}
