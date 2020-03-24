package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
)

func CalculateFileHash(file multipart.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", h.Sum(nil))

	return hash, nil
}
