package api

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims extends RegisteredClaims with a Role field.
type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

// GenerateToken creates a signed JWT with the given user ID, role, and expiry.
func GenerateToken(secret string, userID string, role string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		Role: role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
