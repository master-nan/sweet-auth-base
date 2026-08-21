package token

import (
	"backend/enum"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestJWTParseRejectsNonCanonicalAndMalformedClaimsWithoutPanic(t *testing.T) {
	config := Config{Issuer: "sweet", SecretKey: "secret"}
	claims := jwt.MapClaims{
		"iss": "sweet", "sub": "1", "type": string(enum.AccessToken),
		"iat": time.Now().Unix(), "nbf": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	}
	nonCanonical := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	nonCanonicalValue, err := nonCanonical.SignedString([]byte(config.SecretKey))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewJWTGenerator().ParseToken(nonCanonicalValue, config); err == nil {
		t.Fatal("expected token without jti/sid to be rejected")
	}
	claims["jti"] = "token-id"
	claims["sid"] = "session-id"

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
