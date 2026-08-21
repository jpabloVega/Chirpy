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

func (cfg *apiConfig) editUser(w http.ResponseWriter, req *http.Request) {
	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "Error getting bearer token")
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	decoder := json.NewDecoder(req.Body)
	user := userMail{}
	err = decoder.Decode(&user)
	if err != nil {
		respondWithError(w, 400, "Error decoding request body")
		return
	}

	passHash, err := auth.HashPassword(user.Password)
	if err != nil {
		respondWithError(w, 400, "Error hashing password")
		return
	}

	is_auth, err := auth.CheckPasswordHash(user.Password, passHash)
	if !is_auth {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	newPassHash, err := auth.HashPassword(user.Password)
	if err != nil {
		respondWithError(w, 400, "Error hashing new password")
	}

	editedUserParams := database.EditUserParams{
		ID:             userID,
		HashedPassword: newPassHash,
		Email:          user.Email,
	}

	editedUser, err := cfg.dbQueries.EditUser(req.Context(), editedUserParams)
	if err != nil {
		respondWithError(w, 400, "Error editing user in database")
		return
	}

	finalUser := NewUser{
		Id:        editedUser.ID,
		CreatedAt: editedUser.CreatedAt,
		UpdatedAt: editedUser.UpdatedAt,
		Email:     editedUser.Email,
	}
	respondWithJSON(w, 200, finalUser)
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
		var expireIn time.Duration
		if user.ExpiresInSeconds <= 0 || user.ExpiresInSeconds > 3600 {
			expireIn = time.Second * 3600
		} else {
			expireIn = time.Second * time.Duration(user.ExpiresInSeconds)
		}
		newJWT, err := auth.MakeJWT(dbUser.ID, cfg.secret, expireIn)
		if err != nil {
			log.Printf("Error creating authentication: %v", err)
			respondWithError(w, 400, "Error creating auth")
			return
		}

		newRefToken := auth.MakeRefreshToken()
		if newRefToken == "" {
			respondWithError(w, 400, "Error genetaring refresh token")
			return
		}

		sixtyDays := 60 * 24 * time.Hour
		refreshTokenParams := database.CreateRefreshTokenParams{
			Token:     newRefToken,
			UserID:    dbUser.ID,
			ExpiresAt: time.Now().UTC().Add(sixtyDays),
		}

		_, err = cfg.dbQueries.CreateRefreshToken(req.Context(), refreshTokenParams)
		if err != nil {
			respondWithError(w, 400, "Error adding refresh token to database")
			return
		}

		userData := NewUserResponse{
			Id:           dbUser.ID,
			CreatedAt:    dbUser.CreatedAt,
			UpdatedAt:    dbUser.UpdatedAt,
			Email:        dbUser.Email,
			Token:        newJWT,
			RefreshToken: newRefToken,
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

type NewUserResponse struct {
	Id           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type userMail struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}
