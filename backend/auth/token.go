package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenExpiry = 7 * 24 * time.Hour // 7 days

// GenerateToken creates a JWT token with a 7-day expiry.
func GenerateToken(secret string) (string, time.Time, error) {
	expiresAt := time.Now().Add(tokenExpiry)

	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   "admin",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, expiresAt, nil
}

// ValidateToken checks if a JWT token is valid and not expired.
func ValidateToken(tokenStr, secret string) error {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(secret), nil
	})
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("token is not valid")
	}

	return nil
}
