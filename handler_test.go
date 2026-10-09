package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
)

var testPhotos = []*Photo{
	{0, 0, "qwerty", "qwerty", 0},
	{1, 0, "asdfgh", "qweqwe", 1},
}

type MyMockStorage struct{}

func (ms *MyMockStorage) GetPhotos(ctx context.Context, userID int) ([]*Photo, error) {
	if userID == 0 {
		return testPhotos, nil
	} else {
		return nil, fmt.Errorf("Not found")
	}
}

func (ms *MyMockStorage) Add(ctx context.Context, p *Photo) error {
	return nil
}

func (ms *MyMockStorage) UpdateRate(ctx context.Context, photoID int, count int) error {
	return nil
}

func TestList(t *testing.T) {
	ms := &MyMockStorage{}
	tmp := NewTmpl()

	h := NewPhotoHandler(ms, tmp)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), authKey, &Session{UserID: 0, ID: "test"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.List(w, req)

	resp := w.Result()

	io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := NewMockStorage(ctrl)
	hh := NewPhotoHandler(st, tmp)

	st.EXPECT().GetPhotos(gomock.Any(), 0).Return(nil, fmt.Errorf("error!!!"))
	req = httptest.NewRequest("GET", "/", nil)
	ctx = context.WithValue(req.Context(), authKey, &Session{UserID: 0, ID: "test"})
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	hh.List(w, req)

	resp = w.Result()

	if resp.StatusCode != 500 {
		t.Errorf("expected status code 500, got %d", resp.StatusCode)
	}

	st.EXPECT().GetPhotos(gomock.Any(), 0).Return(nil, nil)
	hhh := NewPhotoHandler(st, tmp)

	hhh.Tmpl.templateIndex, _ = template.New("bad").Parse("{{.NotExist}}")

	req = httptest.NewRequest("GET", "/", nil)
	ctx = context.WithValue(req.Context(), authKey, &Session{UserID: 0, ID: "test"})
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	hhh.List(w, req)

	resp = w.Result()

	if resp.StatusCode != 500 {
		t.Errorf("expected status code 500, got %d", resp.StatusCode)
	}

}
