package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"io/fs"
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

func main() {
	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./images"))
	mux.HandleFunc("/", mainPage)
	mux.HandleFunc("/upload", upload)
	mux.Handle("/images/", http.StripPrefix("/images", fs))

	log.Fatal(http.ListenAndServe(":8080", mux))
}

type Image struct {
	Path string
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./index.html"))
	images := []Image{}
	err := filepath.WalkDir("./images", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), "_160.jpg") {
			imgPath := "/images/" + d.Name()
			images = append(images, Image{Path: imgPath})
			fmt.Printf("Найден файл: %s -> URL: %s\n", d.Name(), imgPath)
		}
		return nil
	})

	if err != nil {
		http.Error(w, "Ошибка чтения изображений", http.StatusBadRequest)
		return
	}

	tmpl.Execute(w, struct {
		Images []Image
	}{
		images,
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

	func() {
		err := generateThumbnails(finalPath, md5Name)
		fmt.Println(err)
	}()

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
