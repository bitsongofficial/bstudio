package bstudio

import (
	"bytes"
	"fmt"
	"github.com/dhowden/tag"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"os"
)

type Upload struct {
	header *multipart.FileHeader
	file   multipart.File
	uid    string
	bs     *BStudio
}

func NewUpload(bs *BStudio, h *multipart.FileHeader, f multipart.File) *Upload {
	return &Upload{
		uid:    uuid.New().String(),
		bs:     bs,
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

func (u *Upload) SavePicture(data []byte) (string, error) {
	return u.bs.sh.Add(bytes.NewReader(data))
}

func (u *Upload) GetMetadata(path string) (map[string]interface{}, error) {
	f, err := os.Open(fmt.Sprintf("%s/original/%s/%s", path, u.GetID(), u.header.Filename))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	metadata, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	trackNr, trackNrOf := metadata.Track()
	discNr, discNrOf := metadata.Disc()

	var picture string
	if metadata.Picture() != nil {
		pictureCid, err := u.SavePicture(metadata.Picture().Data)
		if err == nil {
			picture = pictureCid
		}
	}

	return map[string]interface{}{
		"title":        metadata.Title(),
		"artist":       metadata.Artist(),
		"track":        trackNr,
		"track_of":     trackNrOf,
		"album":        metadata.Album(),
		"album_artist": metadata.AlbumArtist(),
		"comment":      metadata.Comment(),
		"composer":     metadata.Composer(),
		"disc":         discNr,
		"disc_of":      discNrOf,
		"file_type":    metadata.FileType(),
		"format":       metadata.Format(),
		"genre":        metadata.Genre(),
		"lyrics":       metadata.Lyrics(),
		"picture":      picture,
		"year":         metadata.Year(),
	}, nil
}
