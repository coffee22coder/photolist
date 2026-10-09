package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

type Photo struct {
	ID      int
	UserID  int
	Path    string
	Comment string
	Rating  int
}

type User struct {
	ID       int
	Login    string
	Password []byte
}

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(db *sql.DB) *DBStorage {
	st := &DBStorage{
		db: db,
	}

	if err := st.migrate(); err != nil {
		log.Println(err)
	}

	return st
}

func (st *DBStorage) migrate() error {
	q := `
		CREATE TABLE IF NOT EXISTS photos (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			path VARCHAR(255) NOT NULL,
			comment VARCHAR(255),
			rating int NOT NULL DEFAULT 0
		)
	`

	_, err := st.db.Exec(q)

	if err != nil {
		return fmt.Errorf("не удалось создать таблицу photos: %w", err)
	}

	_, err = st.db.Exec(`CREATE INDEX idx_user_id ON photos(user_id)`)
	if err != nil {
		log.Printf("Примечание: индекс idx_user_id: %v", err)
	}

	// al := `ALTER TABLE photos
	// 		ADD COLUMN IF NOT EXISTS comment VARCHAR(255),
	// 		ADD COLUMN IF NOT EXISTS rating int NOT NULL DEFAULT 0;
	// `

	// _, err = st.db.Exec(al)
	// if err != nil {
	// 	log.Printf("Примечание: индекс idx_user_id: %v", err)
	// }

	q = `
		CREATE TABLE IF NOT EXISTS users (
			id int AUTO_INCREMENT PRIMARY KEY,
			login VARCHAR(255) NOT NULL UNIQUE,
			password VARBINARY(255) NOT NULL
		)
	`

	_, err = st.db.Exec(q)

	if err != nil {
		return fmt.Errorf("не удалось создать таблицу users: %w", err)
	}

	_, err = st.db.Exec(`CREATE INDEX idx_login ON users(login)`)
	if err != nil {
		log.Printf("Примечание: индекс idx_login: %v", err)
	}

	q = `
	CREATE TABLE IF NOT EXISTS sessions (
		id VARCHAR(32) NOT NULL PRIMARY KEY,
		user_id int NOT NULL
	)
`
	_, err = st.db.Exec(q)

	if err != nil {
		return fmt.Errorf("не удалось создать таблицу sessions: %w", err)
	}

	log.Println("✅ Миграция БД завершена успешно")

	return nil

}

func (st *DBStorage) Add(ctx context.Context, p *Photo) error {
	op := "storage.add"

	res, err := st.db.ExecContext(ctx,
		`INSERT INTO photos(user_id, path, comment) VALUES(?, ?, ?)`,
		p.UserID, p.Path, p.Comment)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if id == 0 {
		return fmt.Errorf("%s: не удалось получить ID вставленной записи", op)
	}

	return nil
}

func (st *DBStorage) GetPhotos(ctx context.Context, userID int) ([]*Photo, error) {
	op := "storage.get_photos"
	photos := make([]*Photo, 0)
	rows, err := st.db.QueryContext(ctx, `SELECT id, path, comment, rating FROM photos WHERE user_id=?`, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, rating int
		var path, comment string

		if err := rows.Scan(&id, &path, &comment, &rating); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		photo := &Photo{
			ID:      id,
			UserID:  userID,
			Path:    path,
			Comment: comment,
			Rating:  rating,
		}

		photos = append(photos, photo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return photos, nil
}

func (st *DBStorage) UpdateRate(ctx context.Context, photoID int, count int) error {
	op := "storage.update_rate"

	_, err := st.db.ExecContext(ctx, `UPDATE photos SET rating = rating + ? WHERE id = ?`, count, photoID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (st *DBStorage) CreateSession(ctx context.Context, userID int) (*Session, error) {
	op := "storage.create_session"

	uuid := strings.ReplaceAll(uuid.New().String(), "-", "")

	_, err := st.db.ExecContext(ctx, `INSERT INTO sessions(id, user_id) VALUES(?, ?)`, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Session{ID: uuid, UserID: userID}, nil
}

func (st *DBStorage) DeleteSession(ctx context.Context, sessionID string) error {
	op := "storage.delete_session"

	_, err := st.db.ExecContext(ctx, `DELETE FROM sessions WHERE id=?`, sessionID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (st *DBStorage) CheckSession(ctx context.Context, sessionID string) (*Session, error) {
	op := "storage.check_session"

	var userID int

	row := st.db.QueryRowContext(ctx, `SELECT user_id FROM sessions WHERE id=?`, sessionID)
	err := row.Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoAuth
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Session{UserID: userID, ID: sessionID}, nil
}

func (st *DBStorage) GetUser(ctx context.Context, login string) (*User, error) {
	op := "storage.get_user"

	var id int
	var password []byte

	row := st.db.QueryRowContext(ctx, `SELECT id, password FROM users WHERE login=?`, login)
	err := row.Scan(&id, &password)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &User{ID: id, Login: login, Password: password}, nil
}

func (st *DBStorage) CreateUser(ctx context.Context, login string, password []byte) (int, error) {
	op := "storage.create_user"

	res, err := st.db.ExecContext(ctx, `INSERT INTO users(login, password) VALUES(?, ?)`, login, password)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if id == 0 {
		return 0, fmt.Errorf("%s: не удалось создать нового пользователя", op)
	}

	return int(id), nil
}
