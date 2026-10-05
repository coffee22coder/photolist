package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type PhotoStorage interface {
	Add(ctx context.Context, p *Photo) error
	GetPhotos(ctx context.Context, userID int) ([]*Photo, error)
}

type PhotolistHandler struct {
	St   PhotoStorage
	Tmpl *Tmpl
}

func NewPhotoHandler(st PhotoStorage, tmpl *Tmpl) *PhotolistHandler {
	return &PhotolistHandler{
		St:   st,
		Tmpl: tmpl,
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

	if err := h.Tmpl.templateIndex.Execute(w, struct {
		Items []*Photo
	}{
		items,
	}); err != nil {
		http.Error(w, "Ошибка шаблона: Execute", http.StatusInternalServerError)
		return
	}
}

func (h *PhotolistHandler) Upload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(50 << 20)

	if err != nil {
		http.Error(w, "Ошибка обработки формы", http.StatusBadRequest)
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

	sess, err := sessionFromContext(r.Context())
	if err != nil {
		http.Error(w, "Непредвиденная Ошибка", http.StatusInternalServerError)
		return
	}

	if err := h.St.Add(r.Context(), &Photo{UserID: sess.UserID, Path: md5Name}); err != nil {
		http.Error(w, "Не удалось добавить новую элемент в хранилище", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/photos", http.StatusFound)
}
