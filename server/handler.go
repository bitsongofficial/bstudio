package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bitsongofficial/bstudio/ds"
	"github.com/bitsongofficial/bstudio/services"
	"github.com/bitsongofficial/bstudio/transcoder"
	"github.com/google/uuid"
	shell "github.com/ipfs/go-ipfs-api"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	_ "github.com/bitsongofficial/bstudio/server/docs"
	"github.com/gorilla/mux"
	httpswagger "github.com/swaggo/http-swagger"
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
	r.HandleFunc("/api/v1/upload/raw", uploadRawHandler(sh)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/{id}/status", uploadStatusHandler(ds)).Methods(methodGET)

	//r.HandleFunc("/api/v1/msg_handler", msgHandler(cdc)).Methods(methodPOST)
	//r.HandleFunc("/ipfs/{cid}", getIpfsGatewayHandler(ipfsNode)).Methods(methodGET)
}

type UploadAudioResp struct {
	ID           string `json:"id"`
	TranscoderID string `json:"transcoder_id"`
	FileName     string `json:"file_name"`
	TrackID      string `json:"track_id"`
}

type UploadRawResp struct {
	CID string `json:"cid"`
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
// @Success 200 {object} server.UploadRawResp
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

		if uploader.GetContentType() == "image/jpeg" {
			filePath = uploader.GetDir() + "/cover_large.jpg"
		}

		if uploader.GetContentType() == "image/png" {
			filePath = uploader.GetDir() + "/cover_large.png"
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

		res := UploadRawResp{
			CID: cid,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

// @Summary Upload and create raw data
// @Description Upload, create and publish to ipfs a raw data
// @Tags upload
// @Produce json
// @Param raw formData string true "Raw data"
// @Success 200 {object} server.UploadRawResp
// @Failure 400 {object} server.ErrorJson "Error"
// @Router /upload/raw [post]
func uploadRawHandler(sh *shell.Shell) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.FormValue("raw")
		cid, err := sh.Add(strings.NewReader(raw))
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Could not add File: %s", err)))
			return
		}

		fmt.Println(fmt.Sprintf("Added raw content IPFS with CID %s\n", cid))

		res := UploadRawResp{
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
