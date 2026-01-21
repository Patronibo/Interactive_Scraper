package service

import (
	"database/sql"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"interactive-scraper/internal/database"
)

type AuthService struct {
	db          *sql.DB
	jwtSecret   []byte
}

func NewAuthService() *AuthService {
	secret := "your-secret-key-change-in-production"
	return &AuthService{
		jwtSecret: []byte(secret),
	}
}

func (s *AuthService) SetDB(db *sql.DB) {
	s.db = db
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(username, password string) (string, error) {
	if s.db == nil {
		return "", errors.New("database not initialized")
	}

	var passwordHash string
	err := s.db.QueryRow(`
		SELECT password_hash 
		FROM users 
		WHERE username = $1
	`, username).Scan(&passwordHash)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("invalid credentials")
		}
		return "", err
	}

	if !database.CheckPasswordHash(password, passwordHash) {
		return "", errors.New("invalid credentials")
	}

	// Generate JWT token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

