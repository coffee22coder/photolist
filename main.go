package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

const PORT = 8080
const IMAGE_PATH = "./images"
const DSN = "root:love@tcp(127.0.0.1:3306)/photolist?charset=utf8mb4&parseTime=true"

func main() {

	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("Ошибка открытия БД: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	fmt.Println("✅ Успешное подключение к MySQL")

	_, err = os.Stat(IMAGE_PATH)
	if os.IsNotExist(err) {
		fmt.Printf("Папка %s не создана:\n", IMAGE_PATH)
		err = os.Mkdir(IMAGE_PATH, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	st := NewDBStorage(db)
	tmpl := NewTmpl()

	ph := NewHandler(st, tmpl)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./images"))
	mux.HandleFunc("/", ph.List)
	mux.HandleFunc("/upload", ph.Upload)
	mux.Handle("/images/", http.StripPrefix("/images", fs))

	fmt.Printf("🚀 Server run on PORT: %d\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), mux))
}
