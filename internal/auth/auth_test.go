package auth

import "testing"

func TestCorrectPassword(t *testing.T) {
	password := "DorianeFerro!"
	testPassword := "DorianeFerro!"
	hash, err1 := HashPassword(password)
	match, err2 := CheckPasswordHash(testPassword, hash)
	if !match || err1 != nil || err2 != nil {
		t.Errorf(`HashPassword("DorianeFerro!") = %v or failed: %v; CheckPasswordHash("DorianeFerro!", HashPassword("DorianeFerro!")) = %v or failed: %v`, hash, err1, match, err2)
	}
}

func TestIncorrectPassword(t *testing.T) {
	password := "DorianeFerro!"
	testPassword := "DorianeFerro"
	hash, err1 := HashPassword(password)
	match, err2 := CheckPasswordHash(testPassword, hash)
	if match || err1 != nil || err2 != nil {
		t.Errorf(`HashPassword("DorianeFerro!") = %v or failed: %v; CheckPasswordHash("DorianeFerro", HashPassword("DorianeFerro!")) = %v or failed: %v`, hash, err1, match, err2)
	}
}
