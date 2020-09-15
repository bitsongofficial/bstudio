package bstudio

import (
	"fmt"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"os"
)

type Upload struct {
	header *multipart.FileHeader
	file   multipart.File
	uid    string
}

func NewUpload(h *multipart.FileHeader, f multipart.File) *Upload {
	return &Upload{
		uid:    uuid.New().String(),
		header: h,
		file:   f,
	}
}

func (u *Upload) GetContentType() string {
	return u.header.Header.Get("Content-Type")
}

func (u *Upload) GetFile() multipart.File {
	return u.file
}

func (u *Upload) GetID() string {
	return u.uid
}

func (u *Upload) GetName() string {
	return u.header.Filename
}

func (u *Upload) GetSize() int64 {
	return u.header.Size
}

func (u *Upload) IsAudio() bool {
	contentType := u.GetContentType()
	return contentType == "audio/aac" ||
		contentType == "audio/wav" ||
		contentType == "audio/mp3" ||
		contentType == "application/octet-stream" ||
		contentType == "audio/mpeg"
}
func (u *Upload) IsImage() bool {
	contentType := u.GetContentType()
	return contentType == "image/jpeg"
}

func (u *Upload) SaveOriginal(path string) error {
	if _, err := os.Stat(fmt.Sprintf("%s/original", path)); os.IsNotExist(err) {
		if err := os.Mkdir(fmt.Sprintf("%s/original", path), os.ModePerm); err != nil {
			return err
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/original/%s", path, u.GetID())); os.IsNotExist(err) {
		if err := os.Mkdir(fmt.Sprintf("%s/original/%s", path, u.GetID()), os.ModePerm); err != nil {
			return err
		}
	}

	path = fmt.Sprintf("%s/original/%s/%s", path, u.GetID(), u.header.Filename)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	defer f.Close()
	io.Copy(f, u.GetFile())

	return nil
}
