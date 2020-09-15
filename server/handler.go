package server

import (
	"encoding/json"
	"fmt"
	"github.com/bitsongofficial/bstudio/bstudio"
	"github.com/bitsongofficial/bstudio/models"
	_ "github.com/bitsongofficial/bstudio/server/docs"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	httpswagger "github.com/swaggo/http-swagger"
	"go.mongodb.org/mongo-driver/bson"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	methodGET  = "GET"
	methodPOST = "POST"
)

// RegisterRoutes registers all HTTP routes with the provided mux router.
func RegisterRoutes(r *mux.Router, bs *bstudio.BStudio) {
	r.PathPrefix("/swagger/").Handler(httpswagger.WrapHandler)
	r.HandleFunc("/api/v1/upload/audio", uploadAudioHandler(bs)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/image", uploadImageHandler(bs)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/manifest", uploadManifestHandler(bs)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/{uid}/status", uploadStatusHandler(bs)).Methods(methodGET)
}

type UploadCidResp struct {
	Uid      string `json:"uid"`
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

		upload := bstudio.NewUpload(header, file)
		log.Info().Str("filename", header.Filename).Msg("handling audio upload...")

		// check if the file is audio
		log.Info().Str("filename", header.Filename).Msg("check if the file is audio")
		if !upload.IsAudio() {
			log.Error().Str("content-type", upload.GetContentType()).Msg("Wrong content type")
			writeJSONResponse(w, http.StatusUnsupportedMediaType, newErrorJson(fmt.Sprintf("Wrong content type: %s", upload.GetContentType())))
			return
		}

		// save original file
		if err := upload.SaveOriginal(bs.HomeDir); err != nil {
			log.Error().Str("filename", header.Filename).Msg("Cannot save the audio file")
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson(fmt.Sprintf("Cannot save the audio file %s", header.Filename)))
			return
		}
		log.Info().Str("filename: ", header.Filename).Msg("stored original file")

		// insert upload to db
		var mUpload models.Upload
		mUpload.Uid = upload.GetID()
		mUpload.Filename = upload.GetName()
		mUpload.Status = models.UPLOAD_STATUS_PENDING
		mUpload.Size = upload.GetSize()
		mUpload.CreatedAt = time.Now()
		mUpload.UpdatedAt = time.Now()

		_, err = bs.Db.InsertOne(bs.Db.UploadCollection, mUpload)
		if err != nil {
			log.Error().Str("filename", header.Filename).Msg("Failed to insert a new record to mongodb")
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson("failed to initialize file"))
			return
		}

		// check file size
		// check duration
		ts := bstudio.NewTranscoder(bs, upload.GetID(), upload.GetName())
		bs.TQueue <- ts

		res := UploadCidResp{
			Uid:      upload.GetID(),
			FileName: header.Filename,
		}

		bz, err := json.Marshal(res)
		if err != nil {
			log.Error().Str("filename", header.Filename).Msg("Failed to encode response")
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson(fmt.Sprintf("failed to encode response: %s", err.Error())))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(bz)
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
func uploadImageHandler(bs *bstudio.BStudio) http.HandlerFunc {
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

		image, err := bstudio.NewImage(file)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson("Failed to create image object"))
			return
		}

		if err := image.Resize(); err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson("Failed to resize image object"))
			return
		}

		// add to ipfs
		imgObj, err := os.Open(image.GetTmpPath())
		cid, err := bs.Add(imgObj)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson("Failed to store image object"))
			return
		}

		// Remove tmp object
		if err := image.Delete(); err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson("Failed to delete image object"))
			return
		}

		res := UploadCidResp{
			Uid: cid,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

// @Summary Upload and create raw data
// @Description Upload, create and publish to ipfs a manifest data
// @Tags upload
// @Produce json
// @Param manifest formData string true "Manifest"
// @Success 200 {object} server.UploadCidResp
// @Failure 400 {object} server.ErrorJson "Error"
// @Router /upload/manifest [post]
func uploadManifestHandler(bs *bstudio.BStudio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		manifest := r.FormValue("manifest")

		uid, err := uuid.NewUUID()
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Could not create a new uid: %s", err)))
			return
		}

		// put manifest into tmp file
		err = ioutil.WriteFile(fmt.Sprintf("/tmp/%s", uid.String()), []byte(manifest), 0644)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Cannot save manifest: %s", err)))
			return
		}

		// get file
		f, err := os.Open(fmt.Sprintf("/tmp/%s", uid.String()))
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Cannot get tmp manifest: %s", err)))
			return
		}
		defer f.Close()

		cid, err := bs.Add(f)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Cannot store manifest: %s", err)))
			return
		}

		res := UploadCidResp{
			Uid: cid,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

// @Summary Get upload status
// @Description Get upload status by ID.
// @Tags upload
// @Produce json
// @Param uid path string true "UID"
// @Success 200 {object} server.UploadStatusResp
// @Failure 400 {object} server.ErrorJson "Failure to parse the id"
// @Failure 404 {object} server.ErrorJson to find the id"
// @Router /upload/{uid}/status [get]
func uploadStatusHandler(bs *bstudio.BStudio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params = mux.Vars(r)
		status, err := bs.Db.FindOne(bs.Db.UploadCollection, bson.M{"uid": params["uid"]})
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, newErrorJson(fmt.Sprintf("Cannot get transcode status: %s", err)))
			return
		}

		bz, err := json.Marshal(status)
		if err != nil {
			writeJSONResponse(w, http.StatusBadRequest, newErrorJson(fmt.Sprintf("failed to encode response: %s", err.Error())))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(bz)
	}
}
