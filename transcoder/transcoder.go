package transcoder

import (
	"bytes"
	"encoding/json"
	"github.com/bitsongofficial/bstudio/ds"
	"github.com/bitsongofficial/bstudio/services"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FFProbeFormat struct {
	ready        bool
	StreamsCount int32   `json:"nb_streams"`
	Format       string  `json:"format_name"`
	Duration     float32 `json:"duration,string"`
}

type Transcoder struct {
	Uploader *services.Uploader
	Format   FFProbeFormat `json:"format"`
	ds       *ds.Ds
}

func NewTranscoder(u *services.Uploader, ds *ds.Ds) *Transcoder {
	return &Transcoder{
		Uploader: u,
		ds:       ds,
		Format: FFProbeFormat{
			ready: false,
		},
	}
}

type UploadStatus struct {
	Id         uuid.UUID `json:"id"`
	Percentage uint      `json:"percentage"`
	Cid        string    `json:"cid"`
}

func (t *Transcoder) Create() error {
	tidBz, err := t.Uploader.ID.MarshalBinary()
	if err != nil {
		return err
	}

	data := UploadStatus{
		Id:         t.Uploader.ID,
		Percentage: 0,
		Cid:        "",
	}
	dataBz, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err = t.ds.SetAndCommit(tidBz, dataBz); err != nil {
		return err
	}

	return nil
}

func (t *Transcoder) Update(percentage uint, cid *string) error {
	tidBz, err := t.Uploader.ID.MarshalBinary()
	if err != nil {
		return err
	}

	dataBz, err := t.ds.Get(tidBz)
	if err != nil {
		return err
	}

	var status UploadStatus
	if err := json.Unmarshal(dataBz, &status); err != nil {
		return err
	}
	status.Percentage = percentage

	if cid != nil {
		status.Cid = *cid
	}

	dataBz, err = json.Marshal(status)
	if err != nil {
		return err
	}

	err = t.ds.SetAndCommit(tidBz, dataBz)
	if err != nil {
		return err
	}

	return err
}

func (a *Transcoder) SplitToSegments() error {
	newName := a.Uploader.GetDir() + "segments/segment%03d.ts"
	m3u8FileName := a.Uploader.GetDir() + "format/list.m3u8"

	cmd := exec.Command(
		"ffmpeg",
		"-i", a.Uploader.GetTmpConvertedFileName(),
		"-ar", "48000", // sample rate
		"-b:a", "320k", // bitrate
		"-hls_time", "5", // 5s for each segment
		"-hls_segment_type", "mpegts", // hls segment type: Output segment files in MPEG-2 Transport Stream format. This is compatible with all HLS versions.
		"-hls_list_size", "0", //  If set to 0 the list file will contain all the segments
		"-hls_base_url", "../segments/",
		"-hls_segment_filename", newName,
		"-vn", m3u8FileName,
	)

	var ffmpegStdErr bytes.Buffer
	cmd.Stderr = &ffmpegStdErr

	err := cmd.Run()
	if err != nil {
		log.Print("FFMpeg error ", err)
		log.Print(string(ffmpegStdErr.Bytes()))

		return err
	}

	return nil
}

type AudioSegment struct {
	Path   string
	Format FFProbeFormat `json:"format"`
}

type AudioSegments []*AudioSegment

func (as *AudioSegment) ffprobe() error {
	cmd := exec.Command(
		"ffprobe",
		"-v",
		"error",
		"-i",
		as.Path,
		"-print_format",
		"json",
		"-show_format",
	)

	var (
		// There are some uneeded information inside StdOut, skip it
		ffprobeStdOut bytes.Buffer
		ffprobeStdErr bytes.Buffer
	)

	cmd.Stdout = &ffprobeStdOut
	cmd.Stderr = &ffprobeStdErr

	err := cmd.Run()
	if err != nil {
		return err
	}

	ffprobeOutput := ffprobeStdOut.Bytes()
	as.Format = FFProbeFormat{
		ready: true,
	}

	err = json.Unmarshal(ffprobeOutput, &as)
	if err != nil {
		return err
	}

	return nil
}

func (as *AudioSegment) GetDuration() (float32, error) {
	if !as.Format.ready {
		err := as.ffprobe()
		if err != nil {
			return float32(0), err
		}

		return as.Format.Duration, nil

	}

	return as.Format.Duration, nil
}

func (a *Transcoder) GetSegments() (AudioSegments, error) {
	var segments AudioSegments

	err := filepath.Walk(a.Uploader.GetDir(), func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".ts") {
			segment := &AudioSegment{
				Path: "./" + path,
			}
			segments = append(segments, segment)
		}

		return nil
	})

	return segments, err
}

func (a *Transcoder) RemoveFiles() error {
	if err := os.Remove(a.Uploader.GetOriginalFilePath()); err != nil {
		return err
	}
	if err := os.Remove(a.Uploader.GetTmpConvertedFileName()); err != nil {
		return err
	}
	return nil
}

func (a *Transcoder) TranscodeToMp3() error {
	cmd := exec.Command(
		"ffmpeg",
		"-i",
		a.Uploader.GetOriginalFilePath(),
		"-acodec",
		"libmp3lame",
		"-ar",
		"48000",
		"-b:a",
		"320k",
		"-y",
		a.Uploader.GetTmpConvertedFileName(),
	)

	var ffmpegStdErr bytes.Buffer
	cmd.Stderr = &ffmpegStdErr

	err := cmd.Run()
	if err != nil {
		log.Print("FFMpeg error ", err)
		log.Print(string(ffmpegStdErr.Bytes()))

		return err
	}

	_, err = ioutil.ReadFile(a.Uploader.GetTmpConvertedFileName())
	if err != nil {
		return err
	}

	return nil
}

func (a *Transcoder) ffprobe() error {
	cmd := exec.Command(
		"ffprobe",
		"-v",
		"error",
		"-i",
		a.Uploader.GetOriginalFilePath(),
		"-print_format",
		"json",
		"-show_format",
	)

	var (
		// There are some uneeded information inside StdOut, skip it
		ffprobeStdOut bytes.Buffer
		ffprobeStdErr bytes.Buffer
	)

	cmd.Stdout = &ffprobeStdOut
	cmd.Stderr = &ffprobeStdErr

	err := cmd.Run()
	if err != nil {
		return err
	}

	ffprobeOutput := ffprobeStdOut.Bytes()
	a.Format = FFProbeFormat{
		ready: true,
	}

	err = json.Unmarshal(ffprobeOutput, &a)
	if err != nil {
		return err
	}

	return nil
}

func (a *Transcoder) GetDuration() (float32, error) {
	if !a.Format.ready {
		err := a.ffprobe()
		if err != nil {
			return float32(0), err
		}

		return a.Format.Duration, nil

	}

	return a.Format.Duration, nil
}
