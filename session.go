package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type SessionManager interface {
	Check(ctx context.Context, r *http.Request) (*Session, error)
	Create(ctx context.Context, userID int, w http.ResponseWriter) error
	DestroyCurrent(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	DestroyAll(ctx context.Context, userID int, w http.ResponseWriter) error
}

var noAuthPaths = map[string]struct{}{
	"/":           {},
	"/user/login": {},
	"/user/reg":   {},
}

type ContextSess string

const authKey ContextSess = "authSess"

func authMiddleware(sm SessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := noAuthPaths[r.URL.Path]; ok {
			if r.URL.Path == "/" {
				session, err := sm.Check(r.Context(), r)
				if err != nil {
					if errors.Is(err, ErrNoAuth) {
						next.ServeHTTP(w, r)
						return
					}
					http.Error(w, "Error CheckSession", http.StatusInternalServerError)
					return
				}

				ctx := context.WithValue(r.Context(), authKey, session)

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		session, err := sm.Check(r.Context(), r)
		if err != nil {
			http.Error(w, "Ошибка авторизации", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), authKey, session)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func sessionFromContext(ctx context.Context) (*Session, error) {
	op := "auth.session_from_ctx"
	session, ok := ctx.Value(authKey).(*Session)
	if !ok {
		return nil, fmt.Errorf("%s: %w", op, ErrNoAuth)
	}
	return session, nil
}
