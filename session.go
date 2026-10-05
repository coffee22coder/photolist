package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

var noAuthPaths = map[string]struct{}{
	"/":           {},
	"/user/login": {},
	"/user/reg":   {},
}

type ContextSess string

const authKey ContextSess = "authSess"

func authMiddleware(st *DBStorage, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := noAuthPaths[r.URL.Path]; ok {
			if r.URL.Path == "/" {
				cookie, err := r.Cookie("session_id")
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
				session, err := st.CheckSession(r.Context(), cookie.Value)
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

		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Ошибка авторизации", http.StatusUnauthorized)
			return
		}

		session, err := st.CheckSession(r.Context(), cookie.Value)
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
