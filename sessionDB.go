package main

import (
	context "context"
	"net/http"
	"time"
)

type SessionStorage interface {
	CheckSession(ctx context.Context, sessionID string) (*Session, error)
	CreateSession(ctx context.Context, userID int) (*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteAllSession(ctx context.Context, userID int) error
}

type SessionDBManager struct {
	st SessionStorage
}

func NewSessionDBManager(st SessionStorage) *SessionDBManager {
	return &SessionDBManager{
		st: st,
	}
}

func (sm *SessionDBManager) Check(ctx context.Context, r *http.Request) (*Session, error) {

	cookie, err := r.Cookie("session_id")
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, ErrNoAuth
		}
		return nil, err
	}

	return sm.st.CheckSession(ctx, cookie.Value)
}

func (sm *SessionDBManager) Create(ctx context.Context, userID int, w http.ResponseWriter) error {

	sess, err := sm.st.CreateSession(ctx, userID)
	if err != nil {
		return err
	}

	cookie := http.Cookie{
		Name:     "session_id",
		Value:    sess.ID,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	}

	http.SetCookie(w, &cookie)

	return nil
}

func (sm *SessionDBManager) DestroyCurrent(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	cookie := http.Cookie{
		Name:     "session_id",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
	}

	http.SetCookie(w, &cookie)

	sess, err := sessionFromContext(r.Context())
	if err != nil {
		return err
	}

	return sm.st.DeleteSession(ctx, sess.ID)
}

func (sm *SessionDBManager) DestroyAll(ctx context.Context, userID int, w http.ResponseWriter) error {
	return sm.st.DeleteAllSession(ctx, userID)
}
