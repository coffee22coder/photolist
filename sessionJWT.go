package main

import (
	context "context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type SessionJWTManager struct {
	Secret []byte
}

type SessionClaims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func NewSessionJWTManager(s []byte) *SessionJWTManager {
	return &SessionJWTManager{
		Secret: s,
	}
}

func (sm *SessionJWTManager) Check(ctx context.Context, r *http.Request) (*Session, error) {
	claims := SessionClaims{}

	cookie, err := r.Cookie("session")
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, ErrNoAuth
		}
		return nil, err
	}
	token := cookie.Value

	_, err = jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
		method, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok || method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return sm.Secret, nil
	})

	if err != nil {
		return nil, ErrNoAuth
	}

	return &Session{
		UserID: claims.UserID,
		ID:     claims.ID,
	}, nil
}

func (sm *SessionJWTManager) Create(ctx context.Context, userID int, w http.ResponseWriter) error {
	exp := time.Now().Add(10 * time.Minute)
	claims := SessionClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(exp.Unix(), 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        randomName(32),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(sm.Secret)
	if err != nil {
		return err
	}

	cookie := http.Cookie{
		Name:     "session",
		Value:    tokenString,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
	}

	http.SetCookie(w, &cookie)

	return nil
}

func (sm *SessionJWTManager) DestroyCurrent(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	cookie := http.Cookie{
		Name:     "session",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
	}

	http.SetCookie(w, &cookie)
	return nil
}
func (sm *SessionJWTManager) DestroyAll(ctx context.Context, userID int, w http.ResponseWriter) error {
	return nil
}
