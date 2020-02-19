package ipfs

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/tyler-smith/go-bip39"
)

func setupPlugins() error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(repoRoot, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

// Creates an IPFS node and returns its coreAPI
func createNode(ctx context.Context, repoPath string) (icore.CoreAPI, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}

	// Attach the Core API to the constructed node
	return coreapi.NewCoreAPI(node)
}

func spawnDefault(ctx context.Context, path string) (icore.CoreAPI, error) {
	if !fsrepo.IsInitialized(path) {
		initialize(ctx)
	}

	if err := setupPlugins(); err != nil {
		return nil, err

	}

	return createNode(ctx, path)
}

func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
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

const (
	repoRoot        = ".bitsongms/ipfs"
	RepoVersion     = "7"
	nBitsForKeypair = 4096
	DatastoreSpec   = `{"mounts":[{"mountpoint":"/blocks","path":"blocks","shardFunc":"/repo/flatfs/shard/v1/next-to-last/2","type":"flatfs"},{"mountpoint":"/","path":"datastore","type":"levelds"}],"type":"mount"}`
)

var bootstrapNodes = []string{
	// IPFS Bootstrapper nodes.
	//"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	//"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	//"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	//"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",

	// IPFS Cluster Pinning nodes
	"/ip4/138.201.67.219/tcp/4001/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
	"/ip4/138.201.67.220/tcp/4001/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
	"/ip4/138.201.68.74/tcp/4001/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
	"/ip4/94.130.135.167/tcp/4001/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",

	// You can add more nodes here, for example, another IPFS node you might have running locally, mine was:
	// "/ip4/127.0.0.1/tcp/4010/p2p/QmZp2fhDLxjYue2RiUvLwT9MWdnbDxam32qYFnGmxZDh5L",
}

type writer struct{}

func (d *writer) Write(p []byte) (n int, err error) { return 0, nil }

func mustDefaultConfig() *config.Config {
	bootstrapPeers, err := config.ParseBootstrapPeers(bootstrapNodes)
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

func identityFromKey(privkey []byte) (config.Identity, error) {
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

func identityKeyFromSeed(seed []byte, bits int) ([]byte, error) {
	hm := hmac.New(sha256.New, []byte("BitSong Media Server seed"))
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

func initialize(ctx context.Context) {
	if _, err := os.Stat(repoRoot); os.IsNotExist(err) {
		os.Mkdir(repoRoot, os.ModePerm)
	}

	// Version file
	f, err := os.Create(filepath.Join(repoRoot, "version"))
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	_, err = f.Write([]byte(RepoVersion))
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	// Datastore Spec
	f, err = os.Create(filepath.Join(repoRoot, "datastore_spec"))
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	_, err = f.Write([]byte(DatastoreSpec))
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	mnemonic, err := createMnemonic(bip39.NewEntropy, bip39.NewMnemonic)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	conf := mustDefaultConfig()
	seed := bip39.NewSeed(mnemonic, "Secret Passphrase")
	identityKey, err := identityKeyFromSeed(seed, nBitsForKeypair)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	identity, err := identityFromKey(identityKey)
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}
	conf.Identity = identity

	if err := fsrepo.Init(repoRoot, conf); err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return
	}

	if err := initializeIpnsKeyspace(ctx, repoRoot, identityKey); err != nil {
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

func initializeIpnsKeyspace(ctx context.Context, repoRoot string, privKeyBytes []byte) error {
	r, err := fsrepo.Open(repoRoot)
	if err != nil { // NB: repo is owned by the node
		return err
	}

	cfg, err := r.Config()
	if err != nil {
		fmt.Println(fmt.Errorf("%v", err))
		return err
	}
	identity, err := identityFromKey(privKeyBytes)
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

func Start(ctx context.Context) {
	fmt.Println("-- Getting an IPFS node running -- ")

	fmt.Println("Spawning node on a temporary repo")
	ipfs, err := spawnDefault(ctx, repoRoot)
	if err != nil {
		panic(fmt.Errorf("failed to spawn ephemeral node: %s", err))
	}

	fmt.Println("\n-- Adding and getting back files & directories --")

	inputBasePath := "./example/"
	inputPathFile := inputBasePath + "test.txt"

	someFile, err := getUnixfsNode(inputPathFile)
	if err != nil {
		panic(fmt.Errorf("Could not get File: %s", err))
	}

	cidFile, err := ipfs.Unixfs().Add(ctx, someFile)
	if err != nil {
		panic(fmt.Errorf("Could not add File: %s", err))
	}

	fmt.Printf("Added file to IPFS with CID %s\n", cidFile.String())

	_, err = ipfs.Unixfs().Get(ctx, path.New("/ipfs/QmeomffUNfmQy76CQGy9NdmqEnnHU9soCexBnGU3ezPHVH"))
	if err != nil {
		panic(fmt.Errorf("Could not get file with CID: %s", err))
	}

	fmt.Println("\n-- Going to connect to a few nodes in the Network as bootstrappers --")

	go connectToPeers(ctx, ipfs, bootstrapNodes)

	exampleCIDStr := "QmUaoioqU7bxezBQZkUcgcSyokatMY71sxsALxQmRRrHrj"

	fmt.Printf("Fetching a file from the network with CID %s\n", exampleCIDStr)
	testCID := icorepath.New(exampleCIDStr)

	_, err = ipfs.Unixfs().Get(ctx, testCID)
	if err != nil {
		panic(fmt.Errorf("Could not get file with CID: %s", err))
	}

	fmt.Printf("done!\n")
}

func connectToPeers(ctx context.Context, ipfs icore.CoreAPI, peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peerstore.PeerInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peerstore.PeerInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peerstore.PeerInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				log.Printf("failed to connect to %s: %s", peerInfo.ID, err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}
