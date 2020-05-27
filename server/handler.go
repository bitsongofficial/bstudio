package server

import (
	"encoding/json"
	"fmt"
	"github.com/bitsongofficial/bstudio/bstudio"
	_ "github.com/bitsongofficial/bstudio/server/docs"
	"github.com/bitsongofficial/bstudio/services"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
	httpswagger "github.com/swaggo/http-swagger"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	methodGET  = "GET"
	methodPOST = "POST"

	MaxAudioLength = 61000
)

// RegisterRoutes registers all HTTP routes with the provided mux router.
func RegisterRoutes(r *mux.Router, bs *bstudio.BStudio) {
	r.PathPrefix("/swagger/").Handler(httpswagger.WrapHandler)
	r.HandleFunc("/api/v1/upload/audio", uploadAudioHandler(bs)).Methods(methodPOST)
	//r.HandleFunc("/api/v1/upload/image", uploadImageHandler(sh)).Methods(methodPOST)
	//r.HandleFunc("/api/v1/upload/manifest", uploadManifestHandler(sh)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/{cid}/status", uploadStatusHandler(bs)).Methods(methodGET)

	//r.HandleFunc("/api/v1/msg_handler", msgHandler(cdc)).Methods(methodPOST)
	//r.HandleFunc("/ipfs/{cid}", getIpfsGatewayHandler(ipfsNode)).Methods(methodGET)
}

type UploadCidResp struct {
	CID      string `json:"cid"`
	FileName string `json:"filename"`
}

type UploadStatusResp struct {
	ID         string `json:"id"`
	Percentage string `json:"percentage"`
}

// @Summary Upload and transcode audio file
// @Description Upload, transcode and publish to ipfs an audio
// @Tags upload
// @Produce json
// @Param file formData file true "Audio file"
// @Success 200 {object} server.UploadCidResp
// @Failure 400 {object} server.ErrorJson "Error"
// @Router /upload/audio [post]
func uploadAudioHandler(bs *bstudio.BStudio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson("file size is greater then 32mb"))
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson("file field is required"))
			return
		}
		defer file.Close()

		upload := bstudio.NewUpload(bs, header, file)
		log.Info().Str("filename", header.Filename).Msg("handling audio upload...")

		// check if the file is audio
		log.Info().Str("filename", header.Filename).Msg("check if the file is audio")
		if !upload.IsAudio() {
			//uploader.RemoveAll()

			log.Error().Str("content-type", upload.GetContentType()).Msg("Wrong content type")
			writeJSONResponse(w, http.StatusUnsupportedMediaType, newErrorJson(fmt.Sprintf("Wrong content type: %s", upload.GetContentType())))
			return
		}

		// save original file
		cid, err := upload.StoreOriginal()
		log.Info().Str("cid: ", cid).Msg("stored file name " + header.Filename)

		if err != nil {
			//uploader.RemoveAll()
			log.Error().Str("filename", header.Filename).Msg("Cannot save audio file.")
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson(fmt.Sprintf("Cannot save audio file %s", header.Filename)))
			return
		}

		// check file size
		// check duration
		ts := bstudio.NewTranscoder(bs, cid)
		bs.TQueue <- ts

		res := UploadCidResp{
			CID:      cid,
			FileName: header.Filename,
		}

		bz, err := json.Marshal(res)
		if err != nil {
			//uploader.RemoveAll()

			log.Error().Str("filename", header.Filename).Msg("Failed to encode response")
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson(fmt.Sprintf("failed to encode response: %s", err.Error())))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bz)
	}
}

// @Summary Upload and create image file
// @Description Upload, create and publish to ipfs an image
// @Tags upload
// @Produce json
// @Param file formData file true "Image file"
// @Success 200 {object} server.UploadCidResp
// @Failure 400 {object} server.ErrorJson "Error"
// @Router /upload/image [post]
func uploadImageHandler(sh *shell.Shell) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson("file size is greater then 5mb"))
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson("file field is required"))
			return
		}
		defer file.Close()

		log.Info().Str("filename", header.Filename).Msg("handling image upload...")
		uploader := services.NewUploader(&file, header)

		// check if the file is image
		log.Info().Str("filename", header.Filename).Msg("check if the file is image")
		if !uploader.IsImage() {
			uploader.RemoveAll()

			log.Error().Str("content-type", uploader.GetContentType()).Msg("Wrong content type")
			writeJSONResponse(w, http.StatusUnsupportedMediaType, newErrorJson(fmt.Sprintf("Wrong content type: %s", uploader.GetContentType())))
			return
		}

		img, _, _ := image.Decode(file)
		large := resize.Thumbnail(500, 500, img, resize.Lanczos3)

		var filePath string
		var fileName string

		if uploader.GetContentType() == "image/jpeg" {
			fileName = "cover_large.jpg"
			filePath = uploader.GetDir() + "/" + fileName
		}

		if uploader.GetContentType() == "image/png" {
			fileName = "cover_large.png"
			filePath = uploader.GetDir() + "/" + fileName
		}

		outl, err := os.Create(filePath)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson("Failed to create tmp file"))
			return
		}
		defer outl.Close()

		// Encode into jpeg http://blog.golang.org/go-image-package
		if uploader.GetContentType() == "image/jpeg" {
			err = jpeg.Encode(outl, large, nil)
		}

		if uploader.GetContentType() == "image/png" {
			err = png.Encode(outl, large)
		}

		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson("Failed to encode image"))
			return
		}

		// Upload to ipfs
		f, err := os.Open(filePath)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson("Failed to read image"))
			return
		}

		cid, err := sh.Add(f)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Could not add File: %s", err)))
			return
		}

		fmt.Println(fmt.Sprintf("Added cover image to IPFS with CID %s\n", cid))

		// Remove original image
		os.Remove(filePath)

		res := UploadCidResp{
			CID:      cid,
			FileName: fileName,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

// @Summary Upload and create raw data
// @Description Upload, create and publish to ipfs a raw data
// @Tags upload
// @Produce json
// @Param manifest formData string true "Manifest"
// @Param audio_cid formData string true "Audio Cid"
// @Param image_cid formData string true "Image Cid"
// @Success 200 {object} server.UploadCidResp
// @Failure 400 {object} server.ErrorJson "Error"
// @Router /upload/manifest [post]
func uploadManifestHandler(sh *shell.Shell) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		manifest := r.FormValue("manifest")
		audioCid := r.FormValue("audio_cid")
		_ = r.FormValue("image_cid")

		uid, err := uuid.NewUUID()
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Could not create a new uid: %s", err)))
			return
		}

		// 1. create tmp dir
		tmpPath := "/tmp/" + uid.String()
		if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
			err = os.MkdirAll(tmpPath, 0755)
			if err != nil {
				writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Could not create dir: %s", err)))
				return
			}
		}

		// 2. save manifest to root tmp dir
		err = ioutil.WriteFile(tmpPath+"/manifest.json", []byte(manifest), 0644)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Cannot save manifest: %s", err)))
			return
		}

		// 3. download node with audio
		sh.Get(audioCid, fmt.Sprintf("/tmp/%s/audio", uid.String()))

		// 4. download node with image
		/*sh.Get(imageCid, fmt.Sprintf("/tmp/%s/image/%s", uid.String(), imageCid))
		f, err := os.Open(fmt.Sprintf("/tmp/%s/image/%s", uid.String(), imageCid))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		buffer := make([]byte, 512)
		_, err = f.Read(buffer)
		if err != nil {
			if err != nil {
				writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Cannot decode content-type: %s", err)))
				return
			}
		}

		// Use the net/http package's handy DectectContentType function. Always returns a valid
		// content-type by returning "application/octet-stream" if no others seemed to match.
		contentType := http.DetectContentType(buffer)

		if contentType == "image/png" {
			os.Rename(fmt.Sprintf("/tmp/%s/image/%s", uid.String(), imageCid), fmt.Sprintf("/tmp/%s/image/500x500.png", uid.String()))
		}

		if contentType == "image/jpeg" {
			os.Rename(fmt.Sprintf("/tmp/%s/image/%s", uid.String(), imageCid), fmt.Sprintf("/tmp/%s/image/500x500.jpg", uid.String()))
		}*/

		// 5. put root node to ipfs
		cid, err := sh.AddDir(tmpPath)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Cannot upload root node to ipfs: %s", err)))
			return
		}

		// 6. remove tmp file
		err = os.RemoveAll(tmpPath)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Cannot remove tmp node: %s", err)))
			return
		}

		/*
			uploadManifest
			1. send manifest, audiohash, imagehash, signatures of manifest
			2. create a new dir
			3. store metadata
			4. download audio hash and place it into ./audio
			5. download image hash and place it into ./images
			6. put dir to ipfs
			7. return cid
		*/

		res := UploadCidResp{
			CID: cid,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

// @Summary Get upload status
// @Description Get upload status by ID.
// @Tags upload
// @Produce json
// @Param cid path string true "CID"
// @Success 200 {object} server.UploadStatusResp
// @Failure 400 {object} server.ErrorJson "Failure to parse the id"
// @Failure 404 {object} server.ErrorJson to find the id"
// @Router /upload/{cid}/status [get]
func uploadStatusHandler(bs *bstudio.BStudio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params = mux.Vars(r)
		res, err := bs.GetTranscodingStatus(params["cid"])
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Cannot get transcode status: %s", err)))
			return
		}

		var status bstudio.TranscodeStatus
		err = json.Unmarshal(res, &status)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)

	}
}
