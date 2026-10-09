package main

import (
	"context"
	"errors"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestStorageAdd(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("can't create mock DB: %v", err)
	}

	defer db.Close()

	st := NewDBStorage(db)

	userID := 1
	path := "test"
	testPhoto := &Photo{UserID: userID, Path: path}

	mock.
		ExpectExec("INSERT INTO photos").
		WithArgs(userID, path).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = st.Add(context.Background(), testPhoto)
	if err != nil {
		t.Errorf("unexpected error, got err: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	mock.
		ExpectExec("INSERT INTO photos").
		WithArgs(userID, path).
		WillReturnError(errors.New("bla bla bla"))

	err = st.Add(context.Background(), testPhoto)
	if err == nil {
		t.Error("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	mock.
		ExpectExec("INSERT INTO photos").
		WithArgs(userID, path).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("bad result")))

	err = st.Add(context.Background(), testPhoto)
	if err == nil {
		t.Error("expected error, got nil")
	}

	mock.
		ExpectExec("INSERT INTO photos").
		WithArgs(userID, path).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = st.Add(context.Background(), testPhoto)
	if err == nil {
		t.Error("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestStorageList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("can't create mock DB: %v", err)
	}

	defer db.Close()

	st := NewDBStorage(db)

	userId := 0

	rows := sqlmock.NewRows(
		[]string{"id", "path"},
	)

	expect := []*Photo{
		{1, userId, "qwerty", "qwerty", 0},
		{2, userId, "asdfgh", "qweqwe", 1},
	}

	for _, item := range expect {
		rows = rows.AddRow(item.ID, item.Path)
	}

	mock.
		ExpectQuery("SELECT id, path FROM photos").
		WithArgs(userId).
		WillReturnRows(rows)

	photos, err := st.GetPhotos(context.Background(), userId)
	if err != nil {
		t.Errorf("unexpected error, got nil")
	}

	if len(photos) != 2 {
		t.Errorf("expected len=2, got : %d", len(photos))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	mock.
		ExpectQuery("SELECT id, path FROM photos").
		WithArgs(userId).
		WillReturnError(errors.New("bla bla bla"))

	photos, err = st.GetPhotos(context.Background(), userId)
	if err == nil {
		t.Errorf("expected error, got err: %v", err)
	}

	if len(photos) != 0 {
		t.Errorf("expected len=0, got : %d", len(photos))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	errScanRows := sqlmock.NewRows(
		[]string{"id"},
	)

	for _, item := range expect {
		rows = errScanRows.AddRow(item.ID)
	}

	mock.
		ExpectQuery("SELECT id, path FROM photos").
		WithArgs(userId).
		WillReturnRows(errScanRows)

	_, err = st.GetPhotos(context.Background(), userId)
	if err == nil {
		t.Errorf("expected error, got err: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	errRowrows := sqlmock.NewRows([]string{"id", "path"}).
		AddRow(1, "qwerty").
		RowError(0, errors.New("row error"))

	mock.
		ExpectQuery("SELECT id, path FROM photos").
		WithArgs(userId).
		WillReturnRows(errRowrows)

	_, err = st.GetPhotos(context.Background(), userId)
	if err == nil {
		t.Errorf("expected error, got err: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

}
