package bstudio

import (
	"github.com/bitsongofficial/bstudio/database"
	shell "github.com/ipfs/go-ipfs-api"
	"io"
)

const (
	maxTranscoderQueue = 1
)

type BStudio struct {
	sh     *shell.Shell
	TQueue chan *Transcoder
	Db     *database.Database
}

func NewBStudio(db *database.Database, sh *shell.Shell) *BStudio {
	return &BStudio{
		Db:     db,
		sh:     sh,
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
func (bs *BStudio) StartTranscoding() {
	for q := range bs.TQueue {
		q.Transcode()
	}
}
func (bs *BStudio) GetTranscodingStatus(cid string) ([]byte, error) {
	//return bs.Ds.Get([]byte(cid))
	return nil, nil
}

func (bs *BStudio) Subscribe() (*shell.PubSubSubscription, error) {
	return bs.sh.PubSubSubscribe("bstudio")
}
