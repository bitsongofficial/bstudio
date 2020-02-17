package services

import shell "github.com/ipfs/go-ipfs-api"

const (
	IPFS_ENDPOINT = "https://ipfs.infura.io:5001"
)

type Ipfs struct {
	*shell.Shell
}

func NewIpfs() *Ipfs {
	return &Ipfs{
		Shell: shell.NewShell(IPFS_ENDPOINT),
	}
}
