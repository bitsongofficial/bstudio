package bstudio

import (
	shell "github.com/ipfs/go-ipfs-api"
	"io"
	"sync"
)

const (
	maxTranscoderQueue = 1
)

type BStudio struct {
	sh     *shell.Shell
	TQueue chan *Transcoder
	Ds     *Ds
}

func NewBStudio(sh *shell.Shell) *BStudio {
	// Create datastore
	ds := NewDs()
	//defer ds.Db.Close()

	return &BStudio{
		sh:     sh,
		Ds:     ds,
		TQueue: make(chan *Transcoder, maxTranscoderQueue),
	}
}

func (bs *BStudio) Add(r io.Reader) (string, error) {
	return bs.sh.Add(r)
}
func (bs *BStudio) AddDir(dir string) (string, error) {
	return bs.sh.AddDir(dir)
}
func (bs *BStudio) Get(cid, output string) error {
	return bs.sh.Get(cid, output)
}
func (bs *BStudio) StartTranscoding(wg *sync.WaitGroup) {
	go func() {
		for q := range bs.TQueue {
			q.Transcode(wg)
		}
	}()
}
func (bs *BStudio) GetTranscodingStatus(cid string) ([]byte, error) {
	return bs.Ds.Get([]byte(cid))
}
