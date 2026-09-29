package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

const PORT = 8080
const IMAGE_PATH = "./images"

func main() {

	_, err := os.Stat(IMAGE_PATH)
	if os.IsNotExist(err) {
		fmt.Printf("Папка %s не создана:\n", IMAGE_PATH)
		err = os.Mkdir(IMAGE_PATH, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	st := NewStorage()
	tmpl := NewTmpl()

	ph := NewHandler(st, tmpl)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./images"))
	mux.HandleFunc("/", ph.List)
	mux.HandleFunc("/upload", ph.Upload)
	mux.Handle("/images/", http.StripPrefix("/images", fs))

	fmt.Printf("Server run on PORT: %d\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), mux))
}
