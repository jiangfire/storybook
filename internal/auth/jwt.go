package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("invalid token")
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret          []byte
	accessDuration  time.Duration
	refreshDuration time.Duration
}

func NewTokenManager(secret string, accessHours, refreshHours int) *TokenManager {
	return &TokenManager{
		secret:          []byte(secret),
		accessDuration:  time.Duration(accessHours) * time.Hour,
		refreshDuration: time.Duration(refreshHours) * time.Hour,
	}
}

func (m *TokenManager) GenerateAccessToken(userID uint, email, role string) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.accessDuration)
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   email,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	return signed, expiresAt, err
}

func (m *TokenManager) GenerateRefreshToken(userID uint, email, role string) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.refreshDuration)
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   email,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	return signed, expiresAt, err
}

func (m *TokenManager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
