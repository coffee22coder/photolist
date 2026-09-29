package main

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

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
