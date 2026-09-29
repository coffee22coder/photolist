package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

type Photo struct {
	ID     int
	UserID int
	Path   string
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
			path VARCHAR(255) NOT NULL
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

	log.Println("✅ Миграция БД завершена успешно")
	return nil

}

func (st *DBStorage) Add(ctx context.Context, p *Photo) error {
	op := "storage.add"

	res, err := st.db.ExecContext(ctx, `INSERT INTO photos(user_id, path) VALUES(?, ?)`, p.UserID, p.Path)
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
	rows, err := st.db.QueryContext(ctx, `SELECT id, path FROM photos WHERE user_id=?`, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var path string

		if err := rows.Scan(&id, &path); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		photo := &Photo{
			ID:     id,
			UserID: userID,
			Path:   path,
		}

		photos = append(photos, photo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return photos, nil
}
