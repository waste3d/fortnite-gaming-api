package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
}

func NewTokenManager(accessSecret, refreshSecret string) *TokenManager {
	return &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
	}
}

func (m *TokenManager) Generate(userID string) (string, string, error) {
	// Access (15 min)
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"type": "access",
	})
	accessToken, err := at.SignedString(m.accessSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh (7 days)
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type": "refresh",
	})
	refreshToken, err := rt.SignedString(m.refreshSecret)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (m *TokenManager) ValidateAccessToken(tokenStr string) (string, error) {
	return m.validate(tokenStr, m.accessSecret)
}

func (m *TokenManager) ValidateRefreshToken(tokenStr string) (string, error) {
	return m.validate(tokenStr, m.refreshSecret)
}

func (m *TokenManager) validate(tokenStr string, secret []byte) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims["sub"].(string), nil
	}
	return "", errors.New("invalid token")
}
