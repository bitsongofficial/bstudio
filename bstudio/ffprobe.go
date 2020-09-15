package bstudio

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

type ffProbeFormat struct {
	StreamsCount int32   `json:"nb_streams"`
	Format       string  `json:"format_name"`
	Duration     float64 `json:"duration,string"`
	Start        float64 `json:"start,string"`
	Size         int64   `json:"size,string"`
}

type ffProbe struct {
	Format ffProbeFormat `json:"format"`
}

func NewFFProbe(path string) (*ffProbe, error) {
	// TODO: if path exist

	cmd := exec.Command(
		"ffprobe",
		"-v",
		"error",
		"-i",
		path,
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
		return &ffProbe{}, err
	}

	ffprobeOutput := ffprobeStdOut.Bytes()
	out := &ffProbe{}
	err = json.Unmarshal(ffprobeOutput, &out)
	if err != nil {
		return &ffProbe{}, err
	}

	return out, err
}

func (f *ffProbe) GetDuration() float64 {
	return f.Format.Duration
}
