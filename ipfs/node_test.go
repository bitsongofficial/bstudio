package ipfs

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreunix"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"

	"github.com/tyler-smith/go-bip39"
)

const (
	repoRoot                              = ".bitsongms2"
	RepoVersion                           = "7"
	nBitsForKeypair                       = 4096
	BootstrapNodeDefault_LeMarcheSerpette = "/ip4/107.170.133.32/tcp/4001/ipfs/QmUZRGLhcKXF1JyuaHgKm23LvqcoMYwtb9jmh8CkP4og3K"
	BootstrapNodeDefault_BrixtonVillage   = "/ip4/139.59.174.197/tcp/4001/ipfs/QmZfTbnpvPwxCjpCG3CXJ7pfexgkBZ2kgChAiRJrTK1HsM"
	BootstrapNodeDefault_Johari           = "/ip4/139.59.6.222/tcp/4001/ipfs/QmRDcEDK9gSViAevCHiE6ghkaBCU7rTuQj4BDpmCzRvRYg"
)

var BootstrapAddressesDefault = []string{
	BootstrapNodeDefault_LeMarcheSerpette,
	BootstrapNodeDefault_BrixtonVillage,
	BootstrapNodeDefault_Johari,
}

func createMnemonic(newEntropy func(int) ([]byte, error), newMnemonic func([]byte) (string, error)) (string, error) {
	entropy, err := newEntropy(128)
	if err != nil {
		return "", err
	}
	mnemonic, err := newMnemonic(entropy)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

func IdentityFromKey(privkey []byte) (config.Identity, error) {
	ident := config.Identity{}
	sk, err := crypto.UnmarshalPrivateKey(privkey)
	if err != nil {
		return ident, err
	}
	skbytes, err := sk.Bytes()
	if err != nil {
		return ident, err
	}
	ident.PrivKey = base64.StdEncoding.EncodeToString(skbytes)

	id, err := peer.IDFromPublicKey(sk.GetPublic())
	if err != nil {
		return ident, err
	}
	ident.PeerID = id.Pretty()
	return ident, nil
}

func IdentityKeyFromSeed(seed []byte, bits int) ([]byte, error) {
	hm := hmac.New(sha256.New, []byte("OpenBazaar seed"))
	hm.Write(seed)
	reader := bytes.NewReader(hm.Sum(nil))
	sk, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, bits, reader)
	if err != nil {
		return nil, err
	}
	encodedKey, err := sk.Bytes()
	if err != nil {
		return nil, err
	}
	return encodedKey, nil
}

type writer struct{}

func (d *writer) Write(p []byte) (n int, err error) { return 0, nil }

func MustDefaultConfig() *config.Config {
	bootstrapPeers, err := config.ParseBootstrapPeers(BootstrapAddressesDefault)
	if err != nil {
		// BootstrapAddressesDefault are local and should never panic
		panic(err)
	}

	conf, err := config.Init(&writer{}, 4096)
	if err != nil {
		panic(err)
	}

	conf.Ipns.RecordLifetime = "168h"
	conf.Ipns.RepublishPeriod = "24h"
	conf.Discovery.MDNS.Enabled = false
	conf.Addresses = config.Addresses{
		Swarm: []string{
			"/ip4/0.0.0.0/tcp/4001",
			"/ip6/::/tcp/4001",
			"/ip4/0.0.0.0/tcp/9005/ws",
			"/ip6/::/tcp/9005/ws",
		},
		API:     []string{""},
		Gateway: []string{"/ip4/127.0.0.1/tcp/4002"},
	}
	conf.Bootstrap = config.BootstrapPeerStrings(bootstrapPeers)

	return conf
}

func initializeIpnsKeyspace(repoRoot string, privKeyBytes []byte) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := fsrepo.Open(repoRoot)
	if err != nil { // NB: repo is owned by the node
		return err
	}

	cfg, err := r.Config()
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return err
	}
	identity, err := IdentityFromKey(privKeyBytes)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return err
	}

	cfg.Identity = identity

	nd, err := core.NewNode(ctx, &core.BuildCfg{Repo: r})
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return err
	}
	defer nd.Close()

	return namesys.InitializeKeyspace(ctx, nd.Namesys, nd.Pinning, nd.PrivateKey)
}

func initialize() {
	f, err := os.Create(path.Join(repoRoot, "version"))
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	_, err = f.Write([]byte(RepoVersion))
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	mnemonic, err := createMnemonic(bip39.NewEntropy, bip39.NewMnemonic)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	conf := MustDefaultConfig()
	seed := bip39.NewSeed(mnemonic, "Secret Passphrase")
	identityKey, err := IdentityKeyFromSeed(seed, nBitsForKeypair)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	identity, err := IdentityFromKey(identityKey)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}
	conf.Identity = identity

	if err := fsrepo.Init(repoRoot, conf); err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	if err := initializeIpnsKeyspace(repoRoot, identityKey); err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = fsrepo.Open(repoRoot)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}
}

func TestNode(t *testing.T) {
	if !fsrepo.IsInitialized(repoRoot) {
		initialize()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r, err := fsrepo.Open(repoRoot)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
	}

	node, err := core.NewNode(ctx, &core.BuildCfg{Online: true, Routing: libp2p.DHTClientOption, Repo: r})
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to start IPFS node: %v", err))
	}

	out := make(chan interface{}, 10)
	adder, err := coreunix.NewAdder(ctx, node.Pinning, node.Blockstore, node.DAG)
	if err != nil {
		t.Fatal(err)
	}
	adder.Out = out

	fileContent := "testfileA"

	/*file := files.NewBytesFile([]byte(fileContent))

	go func() {
		defer close(out)
		_, _ = adder.AddAllAndPin(file)
	}()

	for o := range out {
		cid := o.(*coreiface.AddEvent).Path.Cid().String()
		require.Equal(t, "QmXcKyuajqj1cWpb31Z8EhVvQZA8JKaQBnWioffhKe7dGV", cid)
	}*/

	bz, err := Cat(node, "QmXcKyuajqj1cWpb31Z8EhVvQZA8JKaQBnWioffhKe7dGV", 10)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	require.Equal(t, fileContent, string(bz))
}
