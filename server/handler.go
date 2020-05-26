package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bitsongofficial/bstudio/ds"
	_ "github.com/bitsongofficial/bstudio/server/docs"
	"github.com/bitsongofficial/bstudio/services"
	"github.com/bitsongofficial/bstudio/transcoder"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	shell "github.com/ipfs/go-ipfs-api"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
	httpswagger "github.com/swaggo/http-swagger"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

const (
	methodGET  = "GET"
	methodPOST = "POST"

	MaxAudioLength = 61000
)

// RegisterRoutes registers all HTTP routes with the provided mux router.
func RegisterRoutes(r *mux.Router, q chan *transcoder.Transcoder, sh *shell.Shell, ds *ds.Ds) {
	r.PathPrefix("/swagger/").Handler(httpswagger.WrapHandler)
	r.HandleFunc("/api/v1/upload/audio", uploadAudioHandler(q, sh, ds)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/image", uploadImageHandler(sh)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/manifest", uploadManifestHandler(sh)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/{id}/status", uploadStatusHandler(ds)).Methods(methodGET)

	//r.HandleFunc("/api/v1/msg_handler", msgHandler(cdc)).Methods(methodPOST)
	//r.HandleFunc("/ipfs/{cid}", getIpfsGatewayHandler(ipfsNode)).Methods(methodGET)
}

type UploadAudioResp struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
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
// @Success 200 {object} server.UploadAudioResp
// @Failure 400 {object} server.ErrorJson "Error"
// @Router /upload/audio [post]
func uploadAudioHandler(q chan *transcoder.Transcoder, sh *shell.Shell, ds *ds.Ds) http.HandlerFunc {
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

		log.Info().Str("filename", header.Filename).Msg("handling audio upload...")
		uploader := services.NewUploader(&file, header)

		// check if the file is audio
		log.Info().Str("filename", header.Filename).Msg("check if the file is audio")
		if !uploader.IsAudio() {
			uploader.RemoveAll()

			log.Error().Str("content-type", uploader.GetContentType()).Msg("Wrong content type")
			writeJSONResponse(w, http.StatusUnsupportedMediaType, newErrorJson(fmt.Sprintf("Wrong content type: %s", uploader.GetContentType())))
			return
		}

		// save original file
		_, err = uploader.SaveOriginal()
		log.Info().Str("filename", header.Filename).Msg("file save original")

		if err != nil {
			uploader.RemoveAll()

			log.Error().Str("filename", uploader.Header.Filename).Msg("Cannot save audio file.")
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson(fmt.Sprintf("Cannot save audio file %s", uploader.Header.Filename)))
			return
		}

		// check file size
		// check duration

		audio := transcoder.NewTranscoder(uploader, ds)
		log.Info().Str("filename", header.Filename).Msg("check audio duration")

		if err := audio.Create(); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson(err.Error()))
			return
		}

		duration, err := audio.GetDuration()
		if err != nil {
			uploader.RemoveAll()

			log.Error().Str("filename", uploader.Header.Filename).Msg(fmt.Sprintf("Cannot get audio duration: %s", err))
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson("Cannot get audio duration"))
			return
		}

		if duration > MaxAudioLength {
			uploader.RemoveAll()

			log.Error().Float32("duration", duration).Msg("File length is too big")
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson("File length is too big"))
			return
		}

		// TODO: Save and publish metadata

		// transcode audio
		log.Info().Str("filename", header.Filename).Msg("transcode audio")
		q <- audio

		res := UploadAudioResp{
			ID:       uploader.ID.String(),
			FileName: uploader.Header.Filename,
		}

		bz, err := json.Marshal(res)
		if err != nil {
			uploader.RemoveAll()

			log.Error().Str("filename", uploader.Header.Filename).Msg("Failed to encode response")
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
// @Param id path string true "ID"
// @Success 200 {object} server.UploadStatusResp
// @Failure 400 {object} server.ErrorJson "Failure to parse the id"
// @Failure 404 {object} server.ErrorJson to find the id"
// @Router /upload/{id}/status [get]
func uploadStatusHandler(ds *ds.Ds) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params = mux.Vars(r)
		id, err := uuid.Parse(params["id"])
		if err != nil {
			return
		}

		tidBz, err := id.MarshalBinary()
		if err != nil {
			return
		}

		data, err := ds.Get(tidBz)
		if err != nil {
			return
		}

		var status transcoder.UploadStatus
		_ = json.Unmarshal(data, &status)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)

	}
}

// TODO: connect shell, add swagger???
func getIpfsGatewayHandler(ipfsNode icore.CoreAPI) http.HandlerFunc {
	// similar to https://github.com/ipfs/go-ipfs/blob/master/core/corehttp/gateway_handler.go
	return func(w http.ResponseWriter, r *http.Request) {
		var params = mux.Vars(r)
		cid := params["cid"]

		fmt.Println(cid)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ipfsCID := icorepath.New(cid)

		data, err := ipfsNode.Unixfs().Get(ctx, ipfsCID)
		if err != nil {
			//writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("cannot serve content"))
			return
		}
		defer data.Close()

		content, ok := data.(files.File)

		if !ok {
			//writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("cannot serve content"))
			return
		}

		defer content.Close()

		w.Header().Set("Cache-Control", "public, max-age=29030400, immutable")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.PathEscape(cid)))
		w.Header().Set("Content-Type", "application/octet-stream")
		//w.Header().Set("Content-Type", "application/x-mpegURL")
		/*w.Header().Set("Content-Length", r.Header.Get("Content-Length"))

		_, err = content.Size()
		if err != nil {
			http.Error(w, "cannot serve files with unknown sizes", http.StatusBadGateway)
			return
		}*/

		io.Copy(w, content)
	}
}
