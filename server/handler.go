package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/bitsongofficial/bitsong-media-server/models"
	"github.com/bitsongofficial/bitsong-media-server/services"
	"github.com/bitsongofficial/bitsong-media-server/transcoder"
	"github.com/bitsongofficial/bitsong-media-server/types"
	"github.com/bitsongofficial/bitsong-media-server/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	_ "github.com/bitsongofficial/bitsong-media-server/server/docs"
	"github.com/gorilla/mux"
	httpswagger "github.com/swaggo/http-swagger"

	"github.com/cosmos/cosmos-sdk/codec"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	methodGET  = "GET"
	methodPOST = "POST"

	MaxAudioLength = 61000
)

// RegisterRoutes registers all HTTP routes with the provided mux router.
func RegisterRoutes(r *mux.Router, q chan *transcoder.Transcoder, ipfsNode icore.CoreAPI, cdc *codec.Codec) {
	r.PathPrefix("/swagger/").Handler(httpswagger.WrapHandler)

	r.HandleFunc("/api/v1/msg_handler", msgHandler(cdc)).Methods(methodPOST)

	r.HandleFunc("/api/v1/upload/audio", uploadAudioHandler(q, cdc)).Methods(methodPOST)
	r.HandleFunc("/api/v1/upload/image", uploadImageHandler(cdc, ipfsNode)).Methods(methodPOST)

	//r.HandleFunc("/api/v1/track_edit", trackEditHandler(cdc)).Methods(methodPOST)
	r.HandleFunc("/api/v1/track", trackHandler(cdc)).Methods(methodPOST)
	r.HandleFunc("/api/v1/tracks", tracksHandler(cdc)).Methods(methodPOST)

	r.HandleFunc("/api/v1/transcode/{id}", getTranscodeHandler()).Methods(methodGET)

	r.HandleFunc("/ipfs/{cid}", getIpfsGatewayHandler(ipfsNode)).Methods(methodGET)
}

type TxReq struct {
	Tx authTypes.StdTx `json:"tx"`
}

func ValidateUploadTx(tx authTypes.StdTx, hash string) (string, error) {
	signers := tx.GetSigners()
	sigs := tx.Signatures

	// Verify signature
	for _, sig := range sigs {
		if !bytes.Equal(sig.Address(), signers[0]) {
			return "", fmt.Errorf("signature does not match signer address")
		}
	}

	for _, msg := range tx.GetMsgs() {
		if msg.Type() == types.TypeMsgUpload {
			uploadMsg := msg.(types.MsgUpload)

			if err := uploadMsg.ValidateBasic(); err != nil {
				return "", fmt.Errorf("failed to validate msg")
			}

			if uploadMsg.FileHash != hash {
				return "", fmt.Errorf("calculated hash does not match file hash")
			}

			if uploadMsg.TrackId != "" {
				return uploadMsg.TrackId, nil
			}

		}
	}

	return "", nil
}

func msgHandler(cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TxReq

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		err = cdc.UnmarshalJSON(body, &req)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request: %w", err))
			return
		}

		signers := req.Tx.GetSigners()
		sigs := req.Tx.Signatures

		// Verify signature
		for _, sig := range sigs {
			if !bytes.Equal(sig.Address(), signers[0]) {
				writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("signature does not match signer address"))
				return
			}
		}

		for _, msg := range req.Tx.GetMsgs() {
			if msg.Type() == types.TypeMsgEditTrack {
				editTrackMsg(w, msg.(types.MsgEditTrack), signers)
				return
			}
		}
	}
}

type EditTrackResp struct {
	TrackID string `json:"track_id"`
}

func editTrackMsg(w http.ResponseWriter, msg types.MsgEditTrack, signers []sdk.AccAddress) {
	if err := msg.ValidateBasic(); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("%s", err.Result().Log))
		return
	}

	trackId, err := primitive.ObjectIDFromHex(msg.TrackId)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to validate track id"))
		return
	}

	track, err := models.GetTrack(trackId)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to get track by trackId"))
		return
	}

	if track.Owner != signers[0].String() {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("signer not authorized to edit"))
		return
	}

	track.Title = msg.Title
	track.Artists = msg.Artists
	track.Featurings = msg.Featurings
	track.Producers = msg.Producers
	track.Genre = msg.Genre
	track.Mood = msg.Mood
	track.ReleaseDate = msg.ReleaseDate
	track.ReleaseDatePrecision = msg.ReleaseDatePrecision
	track.Tags = msg.Tags
	track.Label = msg.Label
	track.Isrc = msg.Isrc
	track.UpcEan = msg.UpcEan
	track.Iswc = msg.Iswc
	track.Credits = msg.Credits
	track.Copyright = msg.Copyright
	track.Visibility = msg.Visibility
	track.Explicit = msg.Explicit
	track.RewardsUsers = msg.RewardsUsers
	track.RewardsPlaylists = msg.RewardsPlaylists
	track.RightsHolders = msg.RightsHolders

	if err := track.Update(); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to update track"))
		return
	}

	bz, err := json.Marshal(track)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to encode response: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(bz)
}

func GetTrackTx(tx authTypes.StdTx) (*models.Track, error) {
	for _, msg := range tx.GetMsgs() {
		if msg.Type() == types.TypeMsgGetTrack {
			getTrackMsg := msg.(types.MsgGetTrack)

			if err := getTrackMsg.ValidateBasic(); err != nil {
				return nil, fmt.Errorf("failed to validate msg")
			}

			trackId, err := primitive.ObjectIDFromHex(getTrackMsg.TrackId)
			if err != nil {
				return nil, fmt.Errorf("failed to validate track id")
			}

			track, err := models.GetTrack(trackId)
			if err != nil {
				return nil, fmt.Errorf("failed to get track by trackId")
			}

			return track, nil
		}
	}

	return nil, fmt.Errorf("no valid msgs")
}

type GetTrackResp struct {
	Track *models.Track `json:"track"`
}

func trackHandler(cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TxReq

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		err = cdc.UnmarshalJSON(body, &req)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request: %w", err))
			return
		}

		// GetTrackTx
		track, err := GetTrackTx(req.Tx)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		res := GetTrackResp{
			Track: track,
		}

		bz, err := json.Marshal(res)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to encode response: %w", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bz)
	}
}

func GetTracksTx(tx authTypes.StdTx) (*[]models.Track, error) {
	signers := tx.GetSigners()
	sigs := tx.Signatures

	// Verify signature
	for _, sig := range sigs {
		if !bytes.Equal(sig.Address(), signers[0]) {
			return nil, fmt.Errorf("signature does not match signer address")
		}
	}

	for _, msg := range tx.GetMsgs() {
		if msg.Type() == types.TypeMsgGetTracks {
			getTracksMsg := msg.(types.MsgGetTracks)

			if err := getTracksMsg.ValidateBasic(); err != nil {
				return nil, fmt.Errorf("failed to validate msg")
			}

			owner := getTracksMsg.FromAddress.String()
			tracks, err := models.GetTracksByOwner(owner)
			if err != nil {
				return nil, fmt.Errorf("failed to get tracks")
			}

			return tracks, nil
		}
	}

	return nil, fmt.Errorf("no valid msgs")
}

type GetTracksResp struct {
	Tracks *[]models.Track `json:"tracks"`
}

func tracksHandler(cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TxReq

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		err = cdc.UnmarshalJSON(body, &req)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request: %w", err))
			return
		}

		// GetTracksTx
		tracks, err := GetTracksTx(req.Tx)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		res := GetTracksResp{
			Tracks: tracks,
		}

		bz, err := json.Marshal(res)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to encode response: %w", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bz)
	}
}

type UploadAudioResp struct {
	ID           string `json:"id"`
	TranscoderID string `json:"transcoder_id"`
	FileName     string `json:"file_name"`
	TrackID      string `json:"track_id"`
}

// @Summary Upload and transcode audio file
// @Description Upload, transcode and publish to ipfs an audio
// @Tags upload
// @Produce json
// @Param file formData file true "Audio file"
// @Success 200 {object} server.UploadAudioResp
// @Failure 400 {object} server.ErrorResponse "Error"
// @Router /upload/audio [post]
func uploadAudioHandler(q chan *transcoder.Transcoder, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TxReq

		file, header, err := r.FormFile("file")
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("file field is required"))
			return
		}
		defer file.Close()

		// Get Tx
		tx := r.FormValue("tx")
		err = cdc.UnmarshalJSON([]byte(tx), &req)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request: %w", err))
			return
		}

		// Calculate File Hash
		hash, err := utils.CalculateFileHash(file)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to calculate sha256"))
			return
		}

		// Validate UploadTx
		_, err = ValidateUploadTx(req.Tx, hash)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		log.Info().Str("filename", header.Filename).Msg("handling audio upload...")
		uploader := services.NewUploader(&file, header)

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

			log.Error().Str("filename", uploader.Header.Filename).Msg(fmt.Sprintf("Cannot get audio duration: %s", err))

			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("Cannot get audio duration"))
			return
		}

		if duration > MaxAudioLength {
			uploader.RemoveAll()

			log.Error().Float32("duration", duration).Msg("File length is too big")

			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("File length is too big"))
			return
		}

		// Create track model
		track := models.NewTrack(header.Filename, req.Tx.GetSigners()[0].String(), duration)
		if err := track.Insert(); err != nil {
			uploader.RemoveAll()

			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		audio.TrackID = track.ID

		// transcode audio
		log.Info().Str("filename", header.Filename).Msg("transcode audio")
		q <- audio

		res := UploadAudioResp{
			ID:           uploader.ID.String(),
			TranscoderID: tm.ID.Hex(),
			FileName:     uploader.Header.Filename,
			TrackID:      track.ID.Hex(),
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
	CID string `json:"cid"`
}

// @Summary Upload and create image file
// @Description Upload, create and publish to ipfs an image
// @Tags upload
// @Produce json
// @Param file formData file true "Image file"
// @Success 200 {object} server.UploadImageResp
// @Failure 400 {object} server.ErrorResponse "Error"
// @Router /upload/image [post]
func uploadImageHandler(cdc *codec.Codec, ipfsNode icore.CoreAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TxReq

		file, header, err := r.FormFile("file")
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("file field is required"))
			return
		}
		defer file.Close()

		// Get Tx
		tx := r.FormValue("tx")
		err = cdc.UnmarshalJSON([]byte(tx), &req)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request: %w", err))
			return
		}

		// Calculate File Hash
		hash, err := utils.CalculateFileHash(file)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("failed to calculate sha256"))
			return
		}

		// Validate UploadTx
		trackIdStr, err := ValidateUploadTx(req.Tx, hash)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		trackId, err := primitive.ObjectIDFromHex(trackIdStr)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		log.Info().Str("filename", header.Filename).Msg("handling image upload...")
		uploader := services.NewUploader(&file, header)

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

		var filePath string

		if uploader.GetContentType() == "image/jpeg" {
			filePath = uploader.GetDir() + "/cover_large.jpg"
		}

		if uploader.GetContentType() == "image/png" {
			filePath = uploader.GetDir() + "/cover_large.png"
		}

		outl, err := os.Create(filePath)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("Failed to create tmp file"))
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
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("Failed to encode image"))
			return
		}

		// TODO: improve image formats
		/*thumbnail := resize.Thumbnail(260, 260, img, resize.Lanczos3)

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
		}*/

		// Upload to ipfs
		listFile, err := utils.GetUnixfsNode(filePath)
		if err != nil {
			panic(fmt.Errorf("Could not get File: %s", err))
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cidFile, err := ipfsNode.Unixfs().Add(ctx, listFile)
		if err != nil {
			panic(fmt.Errorf("Could not add File: %s", err))
		}

		fmt.Println(fmt.Sprintf("Added cover image to IPFS with CID %s\n", cidFile.String()))

		// Remove original image
		os.Remove(filePath)

		// Update track
		track, err := models.GetTrack(trackId)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("Failed to get track by trackId"))
		}

		track.Image = cidFile.String()
		err = track.Update()
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("Failed to update track"))
		}

		res := UploadImageResp{
			CID: cidFile.String(),
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

// TODO: add swagger ??
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
