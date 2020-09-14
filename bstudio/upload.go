package bstudio

import "mime/multipart"

type Upload struct {
	header *multipart.FileHeader
	file   multipart.File
	bs     *BStudio
	uid    string
}

func NewUpload(bs *BStudio, h *multipart.FileHeader, f multipart.File, uid string) *Upload {
	return &Upload{
		uid:    uid,
		header: h,
		file:   f,
		bs:     bs,
	}
}

func (u *Upload) GetContentType() string {
	return u.header.Header.Get("Content-Type")
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

func (u *Upload) StoreOriginal() (string, error) {
	return u.bs.Add(u.file)
}
