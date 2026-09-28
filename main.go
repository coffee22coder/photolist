package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

type Photo struct {
	ID   int
	User int
	Path string
}

const PORT = 8080
const IMAGE_PATH = "./images"

type Storage struct {
	data []Photo
}

var storage Storage

func main() {

	_, err := os.Stat(IMAGE_PATH)
	if os.IsNotExist(err) {
		fmt.Println("Папка %s не создана:", IMAGE_PATH)
		err = os.Mkdir(IMAGE_PATH, 0755)
		if err != nil {
			log.Fatal(err)
		}

	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./images"))
	mux.HandleFunc("/", mainPage)
	mux.HandleFunc("/upload", upload)
	mux.Handle("/images/", http.StripPrefix("/images", fs))

	fmt.Printf("Server run on PORT: %d", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), mux))
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./index.html"))
	photos := []Photo{}

	for _, item := range storage.data {
		thumbURL := "/images/" + item.Path + "_160.jpg"

		localThumbPath := filepath.Join("./images", item.Path+"_160.jpg")
		_, err := os.Stat(localThumbPath)
		if os.IsNotExist(err) {
			fmt.Println("Превью не найдено:", localThumbPath)
			continue
		}

		photos = append(photos, Photo{
			ID:   0,
			User: 0,
			Path: thumbURL,
		})

	}

	tmpl.Execute(w, struct {
		Photos []Photo
	}{
		photos,
	})
}

func upload(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Ошибка получения файла", http.StatusBadRequest)
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

	storage.data = append(storage.data, Photo{Path: md5Name})

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func randomName(l int) string {
	alphabetAndDigits := []rune{
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
		'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
		'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	}

	var builder strings.Builder

	for range l {
		r := rand.Intn(len(alphabetAndDigits))
		builder.WriteRune(alphabetAndDigits[r])
	}

	return builder.String()
}

func generateThumbnails(path string, name string) error {
	sizes := []int{80, 160, 320}

	img, err := imaging.Open(path)
	if err != nil {
		return err
	}

	for _, s := range sizes {
		thumbnail := imaging.Resize(img, s, 0, imaging.Lanczos)
		thumbpPath := filepath.Join("./images", fmt.Sprintf("%s_%d.jpg", name, s))

		err := imaging.Save(thumbnail, thumbpPath)
		if err != nil {
			return fmt.Errorf("Error imaging save: %v", err)
		}
	}

	return nil
}
