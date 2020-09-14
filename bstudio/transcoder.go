package bstudio

import (
	"bytes"
	"fmt"
	"github.com/bitsongofficial/bstudio/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
)

type Transcoder struct {
	bs     *BStudio
	cid    string
	mp3Cid string
	id     primitive.ObjectID
}
type TranscodeResult struct {
	mp3Cid string
	hlsCid string
}

type TranscodeStatus struct {
	Cid        string `json:"cid"`
	HlsCid     string `json:"hls_cid"`
	Percentage uint   `json:"percentage"`
}

func NewTranscoder(bs *BStudio, cid string, id primitive.ObjectID) *Transcoder {
	return &Transcoder{bs: bs, cid: cid, id: id}
}

func (t *Transcoder) GetCidDuration() (float32, error) {
	tmpPath, err := t.getCid()
	if err != nil {
		return 0, err
	}

	ffprobe, err := NewFFProbe(*tmpPath)
	if err != nil {
		return 0, err
	}

	return ffprobe.GetDuration(), err
}
func (t *Transcoder) Transcode() (*TranscodeResult, error) {
	// transcode to mp3
	cid, err := t.transcodeCidToMp3()
	if err != nil {
		return &TranscodeResult{}, err
	}
	t.mp3Cid = cid

	// generate hls
	cid, err = t.transcodeMp3ToHls()
	if err != nil {
		return &TranscodeResult{}, err
	}

	return &TranscodeResult{
		mp3Cid: t.mp3Cid,
		hlsCid: cid,
	}, err
}

func (t *Transcoder) updateStatus(percentage uint, hlsCid string) error {
	var mUpload models.Upload
	mUpload.Percentage = percentage
	if percentage < 100 {
		mUpload.Status = "processing"
	} else {
		mUpload.Status = "completed"
	}
	if hlsCid != "" {
		mUpload.HlsCid = hlsCid
	}
	mUpload.UpdatedAt = time.Now()

	data, err := t.bs.Db.DecodeStruct(mUpload)
	if err != nil {
		return err
	}

	if err := t.bs.Db.UpdateOne(t.bs.Db.UploadCollection, t.id, data); err != nil {
		panic(err)
	}

	return nil
}

func (t *Transcoder) getCid() (*string, error) {
	tmpPath := fmt.Sprintf("/tmp/%s", t.cid)
	err := t.bs.Get(t.cid, tmpPath)
	if err != nil {
		return nil, err
	}

	return &tmpPath, err
}
func (t *Transcoder) transcodeCidToMp3() (string, error) {
	tmpPath, err := t.getCid()
	if err != nil {
		return "", err
	}

	if err := t.updateStatus(5, ""); err != nil {
		panic(err)
	}

	outTmpPath := *tmpPath + ".mp3"

	cmd := exec.Command(
		"ffmpeg",
		"-i",
		*tmpPath,
		"-acodec",
		"libmp3lame",
		"-ar",
		"48000",
		"-b:a",
		"320k",
		"-y",
		outTmpPath,
	)

	var ffmpegStdErr bytes.Buffer
	cmd.Stderr = &ffmpegStdErr

	err = cmd.Run()
	if err != nil {
		//log.Print("FFMpeg error ", err)
		//log.Print(string(ffmpegStdErr.Bytes()))

		return "", err
	}

	_, err = ioutil.ReadFile(outTmpPath)
	if err != nil {
		return "", err
	}

	f, _ := os.Open(outTmpPath)

	if err := t.updateStatus(30, ""); err != nil {
		panic(err)
	}

	return t.bs.Add(f)
}
func (t *Transcoder) transcodeMp3ToHls() (string, error) {
	tmpMp3Path := fmt.Sprintf("/tmp/%s.mp3", t.cid)
	// TODO: check if file exist

	// create tmp hls dir
	tmpHlsPath := fmt.Sprintf("/tmp/%s-hls", t.cid)
	if _, err := os.Stat(tmpHlsPath); os.IsNotExist(err) {
		err = os.MkdirAll(tmpHlsPath, 0755)
		if err != nil {
			return "", err
		}
	}

	segmentFilePattern := tmpHlsPath + "/segment%03d.ts"
	m3u8FileName := tmpHlsPath + "/playlist.m3u8"

	if err := t.updateStatus(40, ""); err != nil {
		panic(err)
	}

	cmd := exec.Command(
		"ffmpeg",
		"-i", tmpMp3Path,
		"-ar", "48000", // sample rate
		"-b:a", "320k", // bitrate
		"-hls_time", "5", // 5s for each segment
		"-hls_segment_type", "mpegts", // hls segment type: Output segment files in MPEG-2 Transport Stream format. This is compatible with all HLS versions.
		"-hls_list_size", "0", //  If set to 0 the list file will contain all the segments
		//"-hls_base_url", "segments/",
		"-hls_segment_filename", segmentFilePattern,
		"-vn", m3u8FileName,
	)

	var ffmpegStdErr bytes.Buffer
	cmd.Stderr = &ffmpegStdErr

	err := cmd.Run()
	if err != nil {
		//log.Print("FFMpeg error ", err)
		//log.Print(string(ffmpegStdErr.Bytes()))

		return "", err
	}

	if err := t.updateStatus(80, ""); err != nil {
		panic(err)
	}

	hlsCid, err := t.bs.AddDir(tmpHlsPath)
	if err != nil {
		return "", err
	}

	if err := t.updateStatus(100, hlsCid); err != nil {
		panic(err)
	}

	return hlsCid, err
}
