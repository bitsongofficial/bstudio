package services

import (
	"fmt"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type Uploader struct {
	ID     uuid.UUID
	File   *multipart.File
	Header *multipart.FileHeader
}

func NewUploader(file *multipart.File, header *multipart.FileHeader) *Uploader {
	id, err := uuid.NewUUID()
	if err != nil {
		panic("cannot generate new uuid")
	}

	return &Uploader{
		ID:     id,
		File:   file,
		Header: header,
	}
}

func (u *Uploader) GetID() string {
	return u.ID.String()
}

func (u *Uploader) GetName() string {
	return u.Header.Filename
}

func (u *Uploader) GetContentType() string {
	return u.Header.Header.Get("Content-Type")
}

func (u *Uploader) GetExtension() string {
	return filepath.Ext(u.Header.Filename)
}

func (u *Uploader) IsAudio() bool {
	contentType := u.GetContentType()
	return contentType == "audio/aac" ||
		contentType == "audio/wav" ||
		contentType == "audio/mp3" ||
		contentType == "application/octet-stream" ||
		contentType == "audio/mpeg"
}

func (u *Uploader) IsImage() bool {
	contentType := u.GetContentType()
	return contentType == "image/jpeg" || contentType == "image/png"
}

func (u *Uploader) GetDir() string {
	dir := os.ExpandEnv(fmt.Sprintf("$HOME/.bstudio/uploader/%s/", u.GetID()))
	u.createDir(dir)

	return dir
}

func (u *Uploader) GetOriginalFilePath() string {
	return u.GetDir() + "original/" + u.GetName()
}

func (u *Uploader) GetTmpConvertedFileName() string {
	return u.GetDir() + "converted" + u.GetExtension()
}

func (u *Uploader) createDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *Uploader) SaveOriginal() (*os.File, error) {
	// create root tmp dir
	if err := u.createDir(u.GetDir()); err != nil {
		return nil, err
	}

	// create original dir
	if err := u.createDir(fmt.Sprintf(`%s%s`, u.GetDir(), "original")); err != nil {
		return nil, err
	}

	// create segments dir
	if err := u.createDir(fmt.Sprintf(`%s%s`, u.GetDir(), "segments")); err != nil {
		return nil, err
	}

	// create format dir
	if err := u.createDir(fmt.Sprintf(`%s%s`, u.GetDir(), "format")); err != nil {
		return nil, err
	}

	// save original file
	buff, err := os.Create(u.GetDir() + "original/" + u.GetName())
	if err != nil {
		return nil, err
	}

	// write the content from POST to the file
	_, err = io.Copy(buff, *u.File)
	if err != nil {
		return nil, err
	}

	return buff, nil
}

func (u *Uploader) RemoveConverted() error {
	return os.RemoveAll(u.GetDir() + "converted.mp3")
}

func (u *Uploader) RemoveAll() error {
	return os.RemoveAll(u.GetDir())
}
