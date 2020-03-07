package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bitsongofficial/bitsong-media-server/models"
	"github.com/bitsongofficial/bitsong-media-server/services"
	"github.com/bitsongofficial/bitsong-media-server/transcoder"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"

	_ "github.com/bitsongofficial/bitsong-media-server/server/docs"
	"github.com/gorilla/mux"
	httpswagger "github.com/swaggo/http-swagger"
)

const (
	methodGET  = "GET"
	methodPOST = "POST"

	MAX_AUDIO_LENGTH = 610
)

// RegisterRoutes registers all HTTP routes with the provided mux router.
func RegisterRoutes(r *mux.Router, q chan *transcoder.Transcoder, ipfsNode icore.CoreAPI) {
	r.PathPrefix("/swagger/").Handler(httpswagger.WrapHandler)

	r.HandleFunc("/api/v1/upload/audio", uploadAudioHandler(q)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/image", uploadImageHandler()).Methods(methodPOST)

	r.HandleFunc("/api/v1/transcode/{id}", getTranscodeHandler()).Methods(methodGET)

	r.HandleFunc("/ipfs/{cid}", getIpfsGatewayHandler(ipfsNode)).Methods(methodGET)
}

type UploadAudioResp struct {
	ID           string  `json:"id"`
	TranscoderID string  `json:"transcoder_id"`
	FileName     string  `json:"file_name"`
	Duration     float32 `json:"duration"`
}

// @Summary Upload and transcode audio file
// @Description Upload, transcode and publish to ipfs an audio
// @Tags upload
// @Produce json
// @Param file formData file true "Audio file"
// @Success 200 {object} server.UploadAudioResp
// @Failure 400 {object} server.ErrorResponse "Error"
// @Router /upload/audio [post]
func uploadAudioHandler(q chan *transcoder.Transcoder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("file")
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("file field is required"))
			return
		}
		defer file.Close()

		log.Info().Str("filename", header.Filename).Msg("handling audio upload...")
		uploader := services.NewUploader(file, header)

		// check if the file is audio
		log.Info().Str("filename", header.Filename).Msg("check if the file is audio")
		if !uploader.IsAudio() {
			uploader.RemoveAll()

			log.Error().Str("content-type", uploader.GetContentType()).Msg("Wrong content type")
			writeErrorResponse(w, http.StatusUnsupportedMediaType, fmt.Errorf("Wrong content type: %s", uploader.GetContentType()))
			return
		}

		// save original file
		_, err = uploader.SaveOriginal()
		log.Info().Str("filename", header.Filename).Msg("file save original")

		if err != nil {
			uploader.RemoveAll()

			log.Error().Str("filename", uploader.Header.Filename).Msg("Cannot save audio file.")

			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("Cannot save audio file %s", uploader.Header.Filename))
			return
		}

		// check file size
		// check duration
		tm := models.NewTranscoder(uploader.ID)
		if err := tm.Create(); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		audio := transcoder.NewTranscoder(uploader, tm.ID)
		log.Info().Str("filename", header.Filename).Msg("check audio duration")

		duration, err := audio.GetDuration()
		if err != nil {
			uploader.RemoveAll()

			log.Error().Str("filename", uploader.Header.Filename).Msg("Cannot get audio duration.")

			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("Cannot get audio duration"))
			return
		}

		if duration > MAX_AUDIO_LENGTH {
			uploader.RemoveAll()

			log.Error().Float32("duration", duration).Msg("File length is too big")

			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("File length is too big"))
			return
		}

		// transcode audio
		log.Info().Str("filename", header.Filename).Msg("transcode audio")

		q <- audio

		res := UploadAudioResp{
			ID:           uploader.ID.String(),
			TranscoderID: tm.ID.Hex(),
			FileName:     uploader.Header.Filename,
			Duration:     duration,
		}

		bz, err := json.Marshal(res)
		if err != nil {
			uploader.RemoveAll()

			log.Error().Str("filename", uploader.Header.Filename).Msg("Failed to encode response")

			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to encode response: %w", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bz)
	}
}

type UploadImageResp struct {
	Filename string `json:"filename"`
}

// @Summary Upload and create image file
// @Description Upload, create and publish to ipfs an image
// @Tags upload
// @Produce json
// @Param file formData file true "Image file"
// @Success 200 {object} server.UploadImageResp
// @Failure 400 {object} server.ErrorResponse "Error"
// @Router /upload/image [post]
func uploadImageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("file")
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("file field is required"))
			return
		}
		defer file.Close()

		log.Info().Str("filename", header.Filename).Msg("handling audio upload...")
		uploader := services.NewUploader(file, header)

		// check if the file is image
		log.Info().Str("filename", header.Filename).Msg("check if the file is image")
		if !uploader.IsImage() {
			uploader.RemoveAll()

			log.Error().Str("content-type", uploader.GetContentType()).Msg("Wrong content type")
			writeErrorResponse(w, http.StatusUnsupportedMediaType, fmt.Errorf("Wrong content type: %s", uploader.GetContentType()))
			return
		}

		img, _, _ := image.Decode(file)

		large := resize.Thumbnail(500, 500, img, resize.Lanczos3)

		outl, err := os.Create(uploader.GetDir() + "/test_large.jpg")
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("Failed to create tmp file"))
			return
		}
		defer outl.Close()

		// Encode into jpeg http://blog.golang.org/go-image-package
		err = jpeg.Encode(outl, large, nil)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("Failed to encode image"))
			return
		}

		thumbnail := resize.Thumbnail(260, 260, img, resize.Lanczos3)

		outt, err := os.Create(uploader.GetDir() + "/test_thumbnail.jpg")
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("Failed to create tmp file"))
			return
		}
		defer outt.Close()

		// Encode into jpeg http://blog.golang.org/go-image-package
		err = jpeg.Encode(outt, thumbnail, nil)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("Failed to encode image"))
			return
		}

		res := UploadImageResp{
			Filename: "test",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

// @Summary Get transcode status
// @Description Get transcode status by ID.
// @Tags transcode
// @Produce json
// @Param id path string true "ID"
// @Success 200 {object} models.Transcoder
// @Failure 400 {object} server.ErrorResponse "Failure to parse the id"
// @Failure 404 {object} server.ErrorResponse "Failure to find the id"
// @Router /transcode/{id} [get]
func getTranscodeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params = mux.Vars(r)
		id := params["id"]

		pid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("cannot decode id"))
			return
		}

		tm := &models.Transcoder{
			ID: pid,
		}

		res, err := tm.Get()
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("id not found"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

// @Summary Get ipfs media content
// @Description Get media content from IPFS by CID.
// @Tags ipfs
// @Produce application/octet-stream
// @Param cid path string true "CID"
// @Success 200 {object} file
// @Failure 400 {object} server.ErrorResponse "Failure to serve media content"
// @Router /ipfs/{cid} [get]
func getIpfsGatewayHandler(ipfsNode icore.CoreAPI) http.HandlerFunc {
	// similar to https://github.com/ipfs/go-ipfs/blob/master/core/corehttp/gateway_handler.go
	return func(w http.ResponseWriter, r *http.Request) {
		var params = mux.Vars(r)
		cid := params["cid"]

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ipfsCID := icorepath.New(cid)

		data, err := ipfsNode.Unixfs().Get(ctx, ipfsCID)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("cannot serve content"))
			return
		}
		defer data.Close()

		content, ok := data.(files.File)

		if !ok {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("cannot serve content"))
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
