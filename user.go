package main

import (
	"bytes"
	"context"
	"net/http"

	"golang.org/x/crypto/argon2"
)

type Session struct {
	UserID int
	ID     string
}

type UserStorage interface {
	GetUser(ctx context.Context, login string) (*User, error)
	CreateUser(ctx context.Context, login string, password []byte) (int, error)
}

type UserHandler struct {
	St   UserStorage
	Tmpl *Tmpl
	Sm   SessionManager
}

func NewUserHandler(st UserStorage, tmpl *Tmpl, sm SessionManager) *UserHandler {
	return &UserHandler{
		St:   st,
		Tmpl: tmpl,
		Sm:   sm,
	}
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		if err := h.Tmpl.templateLogin.Execute(w, struct{}{}); err != nil {
			http.Error(w, "Ошибка шаблона: Execute", http.StatusInternalServerError)
			return
		}
		return
	}

	login := r.PostFormValue("login")
	pass := r.PostFormValue("password")

	if login == "" || pass == "" {
		http.Error(w, "Неверный логин или пароль", http.StatusBadRequest)
		return
	}

	user, err := h.St.GetUser(r.Context(), login)
	if err != nil {
		http.Error(w, "Неверный логин или пароль", http.StatusBadRequest)
		return
	}

	salt := string(user.Password[:8])

	b := hashPassword(pass, salt)
	if !bytes.Equal(b, user.Password) {
		http.Error(w, "Неверный пароль", http.StatusBadRequest)
		return
	}

	err = h.Sm.Create(r.Context(), user.ID, w)
	if err != nil {
		http.Error(w, "Непредвиденная Ошибка", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/photos", http.StatusFound)
}

func (h *UserHandler) Registration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		if err := h.Tmpl.templateReg.Execute(w, struct{}{}); err != nil {
			http.Error(w, "Ошибка шаблона: Execute", http.StatusInternalServerError)
			return
		}
		return
	}

	login := r.PostFormValue("login")
	pass := r.PostFormValue("password")

	if login == "" || pass == "" {
		http.Error(w, "Неверный логин или пароль", http.StatusBadRequest)
		return
	}

	salt := randomName(8)

	psw := hashPassword(pass, salt)

	userID, err := h.St.CreateUser(r.Context(), login, psw)
	if err != nil {
		http.Error(w, "Неверный логин или пароль", http.StatusBadRequest)
		return
	}

	err = h.Sm.Create(r.Context(), userID, w)
	if err != nil {
		http.Error(w, "Непредвиденная Ошибка", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/photos", http.StatusFound)
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {

	err := h.Sm.DestroyCurrent(r.Context(), w, r)
	if err != nil {
		http.Error(w, "Непредвиденная Ошибка", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/user/login", http.StatusFound)

}

func hashPassword(password, salt string) []byte {
	saltByte := []byte(salt)
	idKey := argon2.IDKey([]byte(password), []byte(salt), 1, 64*1024, 4, 32)
	return append(saltByte, idKey...)
}
