package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("failed to create hash: %w", err)
	}
	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()),
		Subject:   userID.String(),
	})

	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", fmt.Errorf("failed to create signed string from tokenSecret: %w", err)
	}
	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse with claims: %w", err)
	}

	UserIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get subject of claims: %w", err)
	}

	UserID, err := uuid.Parse(UserIDString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse user ID string to uuid: %w", err)
	}

	return UserID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authorizationHeader := headers.Get(`Authorization`)
	authorizationHeader = strings.TrimSpace(authorizationHeader)
	authorizationHeader = strings.TrimPrefix(authorizationHeader, "Bearer")
	authorizationHeader = strings.TrimSpace(authorizationHeader)

	if len(authorizationHeader) == 0 {
		return "", fmt.Errorf("failed to find an authorization in the request headers")
	}

	return authorizationHeader, nil
}

func MakeRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to create a random 32 bytes slice")
	}
	return hex.EncodeToString(b), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authorizationHeader := headers.Get(`Authorization`)
	authorizationHeader = strings.TrimSpace(authorizationHeader)
	authorizationHeader = strings.TrimPrefix(authorizationHeader, "ApiKey")
	authorizationHeader = strings.TrimSpace(authorizationHeader)

	if len(authorizationHeader) == 0 {
		return "", fmt.Errorf("failed to find an api key in the request headers")
	}

	return authorizationHeader, nil
}
