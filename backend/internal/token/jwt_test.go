package token

import (
	"backend/enum"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestJWTParseRejectsWrongAlgorithmAndMalformedClaimsWithoutPanic(t *testing.T) {
	config := Config{Issuer: "sweet", SecretKey: "secret"}
	claims := jwt.MapClaims{
		"iss": "sweet", "sub": "1", "type": string(enum.AccessToken),
		"iat": time.Now().Unix(), "nbf": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	}
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	legacyValue, err := legacy.SignedString([]byte(config.SecretKey))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := NewJWTGenerator().ParseToken(legacyValue, config)
	if err != nil || parsed.TokenID != "" {
		t.Fatalf("legacy token compatibility: claims=%+v err=%v", parsed, err)
	}

	wrong := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := wrong.SignedString([]byte(config.SecretKey))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewJWTGenerator().ParseToken(value, config); err == nil {
		t.Fatal("expected wrong signing method rejection")
	}

	claims["sub"] = 1
	malformed := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	value, err = malformed.SignedString([]byte(config.SecretKey))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewJWTGenerator().ParseToken(value, config); err == nil {
		t.Fatal("expected malformed subject rejection")
	}
}
