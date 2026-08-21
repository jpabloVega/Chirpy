package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const TokenTypeAccess = "chirpy-access"

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	key := []byte(tokenSecret)
	currTime := jwt.NewNumericDate(time.Now().UTC())
	expireTime := jwt.NewNumericDate(time.Now().UTC().Add(expiresIn))
	claims := jwt.RegisteredClaims{
		Issuer:    TokenTypeAccess,
		IssuedAt:  currTime,
		ExpiresAt: expireTime,
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(key)
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) { return []byte(tokenSecret), nil })
	if err != nil {
		return uuid.Nil, err
	}

	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, err
	}
	if issuer != string(TokenTypeAccess) {
		return uuid.Nil, errors.New("Invalid issuer")
	}

	id, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("Invalid user ID: %v", err)
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return authHeader, errors.New("Authentication headers not found")
	}
	return strings.ReplaceAll(authHeader, "Bearer ", ""), nil
}

func MakeRefreshToken() string {
	emptyByte := make([]byte, 32)
	_, err := rand.Read(emptyByte)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(emptyByte)
}
