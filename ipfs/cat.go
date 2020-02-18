package ipfs

import (
	"context"
	"errors"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"io/ioutil"
	"strings"
	"time"

	"github.com/ipfs/interface-go-ipfs-core/path"
)

func Cat(n *core.IpfsNode, pth string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !strings.HasPrefix(pth, "/ipfs/") {
		pth = "/ipfs/" + pth
	}
	api, err := coreapi.NewCoreAPI(n)
	if err != nil {
		return nil, err
	}
	nd, err := api.Unixfs().Get(ctx, path.New(pth))
	if err != nil {
		return nil, err
	}

	r, ok := nd.(files.File)
	if !ok {
		return nil, errors.New("Received incorrect type from Unixfs().Get()")
	}

	return ioutil.ReadAll(r)
}
