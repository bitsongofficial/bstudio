package bstudio

import (
	"fmt"
	uuid2 "github.com/google/uuid"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io"
	"os"
)

type Img struct {
	img     image.Image
	tmpPath string
}

func NewImage(r io.Reader) (*Img, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	uuid, err := uuid2.NewUUID()
	if err != nil {
		return nil, err
	}

	return &Img{
		img:     img,
		tmpPath: fmt.Sprintf("/tmp/%s", uuid.String()),
	}, nil
}

func (i *Img) Resize() error {
	tmp, err := os.Create(i.tmpPath)
	if err != nil {
		return err
	}
	defer tmp.Close()

	size := resize.Thumbnail(500, 500, i.img, resize.Lanczos3)
	err = jpeg.Encode(tmp, size, nil)
	if err != nil {
		return err
	}

	return nil
}

func (i *Img) Delete() error {
	return os.Remove(i.tmpPath)
}

func (i *Img) GetTmpPath() string {
	return i.tmpPath
}
