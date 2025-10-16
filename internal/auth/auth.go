package auth

import (
	"fmt"

	"github.com/alexedwards/argon2id"
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
