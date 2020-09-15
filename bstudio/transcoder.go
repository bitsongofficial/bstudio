package bstudio

import (
	"bytes"
	"fmt"
	"github.com/bitsongofficial/bstudio/database"
	"github.com/bitsongofficial/bstudio/models"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func generatePath(homedir, prefix, uid, filename string) string {
	return fmt.Sprintf("%s/%s/%s/%s", homedir, prefix, uid, filename)
}

type Transcoder struct {
	bs       *BStudio
	uid      string
	filename string
}

func NewTranscoder(bs *BStudio, uid, filename string) *Transcoder {
	return &Transcoder{bs: bs, uid: uid, filename: filename}
}

func (t *Transcoder) GetDuration(mp3Path string) (float64, error) {
	ffprobe, err := NewFFProbe(mp3Path)
	if err != nil {
		return 0, err
	}

	return ffprobe.GetDuration(), err
}

func (t *Transcoder) Transcode() error {
	// transcode to mp3
	mp3Path, err := t.transcodeToMp3()
	if err != nil {
		return err
	}

	// get and save duration
	duration, err := t.GetDuration(*mp3Path)
	if err != nil {
		log.Error().Str("get duration failed: ", t.uid).Msg(err.Error())
		return err
	}

	// generate hls
	err = t.transcodeMp3ToHls(*mp3Path)
	if err != nil {
		return err
	}

	cid, err := t.bs.AddDir(fmt.Sprintf("%s/hls/%s", t.bs.HomeDir, t.uid))
	if err != nil {
		log.Error().Str("add dir failed: ", t.uid).Msg(err.Error())
		return err
	}

	var mUpload models.Upload
	mUpload.Duration = duration
	mUpload.Hls = cid
	mUpload.UpdatedAt = time.Now()
	data, err := database.DecodeStruct(mUpload)
	if err != nil {
		return err
	}

	if err := t.bs.Db.UpdateOne(t.bs.Db.UploadCollection, t.uid, data); err != nil {
		panic(err)
	}

	if err := t.updateStatus(100); err != nil {
		return err
	}

	return nil
}

func (t *Transcoder) updateStatus(percentage uint) error {
	var mUpload models.Upload
	mUpload.Percentage = percentage
	if percentage < 100 {
		mUpload.Status = "processing"
	} else {
		mUpload.Status = "completed"
	}
	mUpload.UpdatedAt = time.Now()

	data, err := database.DecodeStruct(mUpload)
	if err != nil {
		return err
	}

	if err := t.bs.Db.UpdateOne(t.bs.Db.UploadCollection, t.uid, data); err != nil {
		panic(err)
	}

	log.Info().Str("update status: ", t.uid).Msg(fmt.Sprintf("%d%%", percentage))

	return nil
}

func (t *Transcoder) transcodeToMp3() (*string, error) {
	if err := t.updateStatus(5); err != nil {
		return nil, err
	}

	// create mp3 dir
	if _, err := os.Stat(fmt.Sprintf("%s/mp3", t.bs.HomeDir)); os.IsNotExist(err) {
		if err := os.Mkdir(fmt.Sprintf("%s/mp3", t.bs.HomeDir), os.ModePerm); err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/mp3/%s", t.bs.HomeDir, t.uid)); os.IsNotExist(err) {
		if err := os.Mkdir(fmt.Sprintf("%s/mp3/%s", t.bs.HomeDir, t.uid), os.ModePerm); err != nil {
			return nil, err
		}
	}

	oldPath := generatePath(t.bs.HomeDir, "original", t.uid, t.filename)
	extension := filepath.Ext(t.filename)
	newFilename := t.filename[0:len(t.filename)-len(extension)] + ".mp3"
	newPath := generatePath(t.bs.HomeDir, "mp3", t.uid, newFilename)

	cmd := exec.Command("ffmpeg",
		"-i", oldPath,
		"-acodec", "libmp3lame",
		"-ar", "48000",
		"-b:a", "320k",
		"-y", newPath,
	)

	var ffmpegStdErr bytes.Buffer
	cmd.Stderr = &ffmpegStdErr

	log.Info().Str("mp3 conversion: ", t.uid).Msg("start conversion")
	err := cmd.Run()
	if err != nil {
		log.Error().Str("transcodeToMp3 failed: ", t.uid).Msg(err.Error())
		return nil, err
	}
	log.Info().Str("mp3 conversion: ", t.uid).Msg("end conversion")

	if err := t.updateStatus(30); err != nil {
		return nil, err
	}

	return &newPath, nil
}
func (t *Transcoder) transcodeMp3ToHls(mp3Path string) error {
	// TODO: check if file exist

	// create hls dir
	if _, err := os.Stat(fmt.Sprintf("%s/hls", t.bs.HomeDir)); os.IsNotExist(err) {
		if err := os.Mkdir(fmt.Sprintf("%s/hls", t.bs.HomeDir), os.ModePerm); err != nil {
			return err
		}
	}
	if _, err := os.Stat(fmt.Sprintf("%s/hls/%s", t.bs.HomeDir, t.uid)); os.IsNotExist(err) {
		if err := os.Mkdir(fmt.Sprintf("%s/hls/%s", t.bs.HomeDir, t.uid), os.ModePerm); err != nil {
			return err
		}
	}

	newPath := generatePath(t.bs.HomeDir, "hls", t.uid, "playlist.m3u8")
	segmentFilePattern := generatePath(t.bs.HomeDir, "hls", t.uid, "segment%03d.ts")

	if err := t.updateStatus(40); err != nil {
		panic(err)
	}

	log.Info().Str("hls conversion: ", t.uid).Msg("start conversion")
	cmd := exec.Command("ffmpeg",
		"-i", mp3Path,
		"-ar", "48000", // sample rate
		"-b:a", "320k", // bitrate
		"-hls_time", "5", // 5s for each segment
		"-hls_segment_type", "mpegts", // hls segment type: Output segment files in MPEG-2 Transport Stream format. This is compatible with all HLS versions.
		"-hls_list_size", "0", //  If set to 0 the list file will contain all the segments
		//"-hls_base_url", "segments/",
		"-hls_segment_filename", segmentFilePattern,
		"-vn", newPath,
	)

	var ffmpegStdErr bytes.Buffer
	cmd.Stderr = &ffmpegStdErr

	err := cmd.Run()
	if err != nil {
		log.Error().Str("transcodeToHls failed: ", t.uid).Msg(err.Error())
		return err
	}
	log.Info().Str("mp3 conversion: ", t.uid).Msg("end conversion")

	if err := t.updateStatus(80); err != nil {
		return err
	}

	return err
}
