package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type TokenManager interface {
	Create(sess *Session, expUnix int64) (string, error)
	Check(sess *Session, token string) (bool, error)
}

type PhotoStorage interface {
	Add(ctx context.Context, p *Photo) error
	GetPhotos(ctx context.Context, userID int) ([]*Photo, error)
	UpdateRate(ctx context.Context, photoID int, count int) error
}

type PhotolistHandler struct {
	St   PhotoStorage
	Tmpl *Tmpl
	Tm   TokenManager
}

func NewPhotoHandler(st PhotoStorage, tmpl *Tmpl, tm TokenManager) *PhotolistHandler {
	return &PhotolistHandler{
		St:   st,
		Tmpl: tmpl,
		Tm:   tm,
	}
}

func (h *PhotolistHandler) Index(w http.ResponseWriter, r *http.Request) {
	_, err := sessionFromContext(r.Context())
	if err != nil {
		http.Redirect(w, r, "/user/login", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/photos/", http.StatusFound)
}

func (h *PhotolistHandler) List(w http.ResponseWriter, r *http.Request) {
	sess, err := sessionFromContext(r.Context())
	if err != nil {
		http.Error(w, "Непредвиденная Ошибка", http.StatusInternalServerError)
		return
	}

	items, err := h.St.GetPhotos(r.Context(), sess.UserID)
	if err != nil {
		http.Error(w, "Ошибка хранилища: GetPhotos", http.StatusInternalServerError)
		return
	}

	now := time.Now().Add(1 * time.Hour).Unix()

	token, err := h.Tm.Create(sess, now)

	if err != nil {
		http.Error(w, "Ошибка генерации токена", http.StatusInternalServerError)
		return
	}

	if err := h.Tmpl.templateIndex.Execute(w, struct {
		Items []*Photo
		Token string
	}{
		items,
		token,
	}); err != nil {
		http.Error(w, "Ошибка шаблона: Execute", http.StatusInternalServerError)
		return
	}
}

func (h *PhotolistHandler) Rate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// if r.Method != http.MethodPost {
	// 	http.Error(w, "Неправильные параметры запроса", http.StatusBadRequest)
	// 	return
	// }

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, `{"err": "bad id"}`, http.StatusBadRequest)
		return
	}

	token := r.Header.Get("CSRF-token")

	if token == "" {
		http.Error(w, `{"err": "bad token"}`, http.StatusBadRequest)
		return
	}

	sess, err := sessionFromContext(r.Context())
	if err != nil {
		http.Error(w, "Непредвиденная Ошибка", http.StatusInternalServerError)
		return
	}

	ok, err := h.Tm.Check(sess, token)
	if err != nil {
		http.Error(w, `{"err": "bad token"}`, http.StatusBadRequest)
		return
	}

	if !ok {
		http.Error(w, `{"err": "bad token"}`, http.StatusBadRequest)
		return
	}

	var count int
	switch r.FormValue("vote") {
	case "up":
		count = 1
	case "down":
		count = -1
	default:
		http.Error(w, `{"err": "bad vote"}`, http.StatusBadRequest)
		return
	}

	err = h.St.UpdateRate(r.Context(), id, count)
	if err != nil {
		http.Error(w, `{"err": "db err"}`, http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]int{"id": id})

}

func (h *PhotolistHandler) Upload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(50 << 20)

	if err != nil {
		http.Error(w, "Ошибка обработки формы", http.StatusBadRequest)
		return
	}

	comment := r.FormValue("comment")

	token := r.FormValue("csrf-token")
	if token == "" {
		http.Error(w, `{"err": "bad token"}`, http.StatusBadRequest)
		return
	}

	sess, err := sessionFromContext(r.Context())
	if err != nil {
		http.Error(w, "Непредвиденная Ошибка", http.StatusInternalServerError)
		return
	}

	ok, err := h.Tm.Check(sess, token)
	if err != nil {
		http.Error(w, `{"err": "bad token"}`, http.StatusBadRequest)
		return
	}

	if !ok {
		http.Error(w, `{"err": "bad token"}`, http.StatusBadRequest)
		return
	}

	tmpName := randomName(8)
	tmpPath := filepath.Join("./images", tmpName)

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		http.Error(w, "Ошибка создания временного файла", http.StatusInternalServerError)
		return
	}
	defer tmpFile.Close()

	hasher := md5.New()

	multiWriter := io.MultiWriter(tmpFile, hasher)

	file, header, err := r.FormFile("my_file")
	if err != nil {
		http.Error(w, "Ошибка получения файла", http.StatusInternalServerError)
		return
	}

	defer file.Close()

	if _, err := io.Copy(multiWriter, file); err != nil {
		os.Remove(tmpPath)
		http.Error(w, "Ошибка записи файла", http.StatusInternalServerError)
		return
	}

	tmpFile.Close()

	hashBytes := hasher.Sum(nil)
	md5Name := hex.EncodeToString(hashBytes)

	ext := filepath.Ext(header.Filename)
	finalPath := filepath.Join("./images", md5Name+ext)

	err = os.Rename(tmpPath, finalPath)
	if err != nil {
		http.Error(w, "Ошибка переименования файла", http.StatusInternalServerError)
		return
	}

	err = generateThumbnails(finalPath, md5Name)
	if err != nil {
		http.Error(w, "Ошибка записи файла", http.StatusInternalServerError)
		return
	}

	if err := h.St.Add(r.Context(), &Photo{UserID: sess.UserID, Path: md5Name, Comment: comment}); err != nil {
		http.Error(w, "Не удалось добавить новую элемент в хранилище", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/photos", http.StatusFound)
}
