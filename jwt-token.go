package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTTokenManager struct {
	Secret []byte
}

func NewJWTTokenManager(s []byte) *JWTTokenManager {
	return &JWTTokenManager{
		Secret: s,
	}
}

type CustomClaims struct {
	UserID    int    `json:"user_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func (tm *JWTTokenManager) Create(sess *Session, expUnix int64) (string, error) {
	claims := CustomClaims{
		UserID:    sess.UserID,
		SessionID: sess.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(expUnix, 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(tm.Secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil

}

func (tm *JWTTokenManager) Check(sess *Session, token string) (bool, error) {
	claims := CustomClaims{}

	_, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
		method, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok || method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return tm.Secret, nil
	})

	if err != nil {
		return false, err
	}

	ok := claims.SessionID == sess.ID && claims.UserID == sess.UserID
	if !ok {
		return false, fmt.Errorf("token invalid")
	}

	return true, nil

}
