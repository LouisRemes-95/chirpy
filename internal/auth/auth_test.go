package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCorrectPassword(t *testing.T) {
	password := "DorianeFerro!"
	testPassword := "DorianeFerro!"
	hash, err1 := HashPassword(password)
	if err1 != nil {
		t.Errorf(`HashPassword("DorianeFerro!") = %v or failed: %v`, hash, err1)
		return
	}
	match, err2 := CheckPasswordHash(testPassword, hash)
	if !match || err2 != nil {
		t.Errorf(`HashPassword("DorianeFerro!") = %v or failed: %v; CheckPasswordHash("DorianeFerro!", HashPassword("DorianeFerro!")) = %v or failed: %v`, hash, err1, match, err2)
	}
}

func TestIncorrectPassword(t *testing.T) {
	password := "DorianeFerro!"
	testPassword := "DorianeFerro"
	hash, err1 := HashPassword(password)
	if err1 != nil {
		t.Errorf(`HashPassword("DorianeFerro!") = %v or failed: %v`, hash, err1)
		return
	}
	match, err2 := CheckPasswordHash(testPassword, hash)
	if match || err2 != nil {
		t.Errorf(`HashPassword("DorianeFerro!") = %v or failed: %v; CheckPasswordHash("DorianeFerro", HashPassword("DorianeFerro!")) = %v or failed: %v`, hash, err1, match, err2)
	}
}

func TestCorrectValidateJWT(t *testing.T) {
	expiresIn, err := time.ParseDuration("10s")
	if err != nil {
		t.Errorf(`failed to parse a duration: %v`, err)
	}

	ID := uuid.New()
	tokenSecret := "testofsecretstring"
	tokenString, err := MakeJWT(ID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf(`failed to make JWT token string: %v`, err)
	}

	ReturnedID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Errorf(`failed to Validate JWT token string: %v`, err)
	}

	if ReturnedID != ID {
		t.Errorf(`Validated JWT token string returned ID = %v, expection ID = %v`, ReturnedID, ID)
	}
}

func TestInCorrectValidateJWT(t *testing.T) {
	expiresIn, err := time.ParseDuration("10s")
	if err != nil {
		t.Errorf(`failed to parse a duration: %v`, err)
	}

	ID := uuid.New()
	tokenSecret := "testofsecretstring"
	tokenString, err := MakeJWT(ID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf(`failed to make JWT token string: %v`, err)
	}

	tokenSecret += "andissues"
	_, err = ValidateJWT(tokenString, tokenSecret)

	if err == nil {
		t.Errorf(`JWT Validation, but different secret token strings for MakeJWT() and ValidateJWT`)
	}
}

func TestExpiredValidateJWT(t *testing.T) {
	expiresIn, err := time.ParseDuration("0s")
	if err != nil {
		t.Errorf(`failed to parse a duration: %v`, err)
	}

	ID := uuid.New()
	tokenSecret := "testofsecretstring"
	tokenString, err := MakeJWT(ID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf(`failed to make JWT token string: %v`, err)
	}

	time.Sleep(1 * time.Millisecond)
	ReturnedID, err := ValidateJWT(tokenString, tokenSecret)

	if ReturnedID == ID || err == nil {
		t.Errorf(`JWT Validation, but should have expired`)
	}
}

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}
	headers.Add("Authorization", " Bearer hello world")
	tokenString, err := GetBearerToken(headers)
	if tokenString != "hello world" || err != nil {
		t.Errorf(`Incorrect token string GetBearerToken("Authorization", "Bearer hello world") = %v or failed with error: %v`, tokenString, err)
	}
}

func TestGetBearerTokenEmptyValue(t *testing.T) {
	headers := http.Header{}
	headers.Add("Authorization", "Bearer ")
	tokenString, err := GetBearerToken(headers)
	if err == nil {
		t.Errorf(`Token string GetBearerToken("Authorization", "Bearer ") = %v, but should fail since token string empty`, tokenString)
	}
}

func TestGetBearerTokenEmptyKey(t *testing.T) {
	headers := http.Header{}
	tokenString, err := GetBearerToken(headers)
	if err == nil {
		t.Errorf(`Token string GetBearerToken("Authorization", "Bearer ") = %v, but should fail since headers empty`, tokenString)
	}
}
