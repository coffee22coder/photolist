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

	fmt.Printf("Server run on PORT: %d", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), mux))
}

// func mainPage(w http.ResponseWriter, r *http.Request) {
// 	tmpl := template.Must(template.ParseFiles("./index.html"))
// 	photos := []Photo{}

// 	for _, item := range storage.data {
// 		thumbURL := "/images/" + item.Path + "_160.jpg"

// 		localThumbPath := filepath.Join("./images", item.Path+"_160.jpg")
// 		_, err := os.Stat(localThumbPath)
// 		if os.IsNotExist(err) {
// 			fmt.Println("Превью не найдено:", localThumbPath)
// 			continue
// 		}

// 		photos = append(photos, Photo{
// 			ID:     0,
// 			UserID: 0,
// 			Path:   thumbURL,
// 		})

// 	}

// 	tmpl.Execute(w, struct {
// 		Photos []Photo
// 	}{
// 		photos,
// 	})
// }

// func upload(w http.ResponseWriter, r *http.Request) {
// 	err := r.ParseMultipartForm(50 << 20)

// 	if err != nil {
// 		http.Error(w, "Ошибка обработки формы", http.StatusBadRequest)
// 		return
// 	}

// 	tmpName := randomName(8)
// 	tmpPath := filepath.Join("./images", tmpName)

// 	tmpFile, err := os.Create(tmpPath)
// 	if err != nil {
// 		http.Error(w, "Ошибка создания временного файла", http.StatusInternalServerError)
// 		return
// 	}
// 	defer tmpFile.Close()

// 	hasher := md5.New()

// 	multiWriter := io.MultiWriter(tmpFile, hasher)

// 	file, header, err := r.FormFile("my_file")
// 	if err != nil {
// 		http.Error(w, "Ошибка получения файла", http.StatusBadRequest)
// 		return
// 	}

// 	defer file.Close()

// 	if _, err := io.Copy(multiWriter, file); err != nil {
// 		os.Remove(tmpPath)
// 		http.Error(w, "Ошибка записи файла", http.StatusInternalServerError)
// 		return
// 	}

// 	tmpFile.Close()

// 	hashBytes := hasher.Sum(nil)
// 	md5Name := hex.EncodeToString(hashBytes)

// 	ext := filepath.Ext(header.Filename)
// 	finalPath := filepath.Join("./images", md5Name+ext)

// 	err = os.Rename(tmpPath, finalPath)
// 	if err != nil {
// 		http.Error(w, "Ошибка переименования файла", http.StatusInternalServerError)
// 		return
// 	}

// 	err = generateThumbnails(finalPath, md5Name)
// 	if err != nil {
// 		http.Error(w, "Ошибка записи файла", http.StatusInternalServerError)
// 		return
// 	}

// 	storage.data = append(storage.data, Photo{Path: md5Name})

// 	http.Redirect(w, r, "/", http.StatusMovedPermanently)
// }
