package ipfs

import (
	"context"
	"github.com/ipfs/go-datastore"
	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreunix"
	coremock "github.com/ipfs/go-ipfs/core/mock"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path"
	"testing"

	syncds "github.com/ipfs/go-datastore/sync"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
)

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	teardown()
	os.Exit(retCode)
}

func setup() {
	os.MkdirAll(path.Join(os.TempDir(), "root"), os.ModePerm)
	d := []byte("hello world")
	ioutil.WriteFile(path.Join(os.TempDir(), "root", "test"), d, os.ModePerm)
}

func teardown() {
	os.RemoveAll(path.Join(os.TempDir(), "root"))
}

func TestAddFile(t *testing.T) {
	n, err := coremock.NewMockNode()
	if err != nil {
		t.Error(err)
	}
	hash, err := AddFile(n, path.Join(os.TempDir(), "root", "test"))
	if err != nil {
		t.Error(err)
	}
	if hash != "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD" {
		t.Error("Ipfs add file failed")
	}
}

const testPeerID = "QmTFauExutTsy4XP6JbMFcw2Wa9645HJt2bTqL6qYDCKfe"

func setupNode() (*core.IpfsNode, error) {
	r := &repo.Mock{
		C: config.Config{
			Identity: config.Identity{
				PeerID: testPeerID, // required by offline node
			},
		},
		D: syncds.MutexWrap(datastore.NewMapDatastore()),
	}
	node, err := core.NewNode(context.Background(), &core.BuildCfg{Repo: r})

	return node, err
}

func TestAdd2File(t *testing.T) {
	node, err := setupNode()
	if err != nil {
		t.Fatal(err)
	}

	out := make(chan interface{}, 10)
	adder, err := coreunix.NewAdder(context.Background(), node.Pinning, node.Blockstore, node.DAG)
	if err != nil {
		t.Fatal(err)
	}
	adder.Out = out

	file := files.NewBytesFile([]byte("testfileA"))

	go func() {
		defer close(out)
		_, _ = adder.AddAllAndPin(file)
		// Ignore errors for clarity - the real bug would be gc'ing files while adding them, not this resultant error
	}()

	for o := range out {
		cid := o.(*coreiface.AddEvent).Path.Cid().String()
		require.Equal(t, "QmXcKyuajqj1cWpb31Z8EhVvQZA8JKaQBnWioffhKe7dGV", cid)
	}
}
