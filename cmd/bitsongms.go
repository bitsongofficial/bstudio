package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bitsongofficial/bitsong-media-server/ipfs"
	"github.com/bitsongofficial/bitsong-media-server/models"
	"github.com/bitsongofficial/bitsong-media-server/transcoder"
	"github.com/bitsongofficial/bitsong-media-server/types"
	"github.com/bitsongofficial/bitsong-media-server/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gorilla/mux"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitsongofficial/bitsong-media-server/server"
)

const (
	logLevelJSON = "json"
	logLevelText = "text"
	dbPath       = ".bitsongms"
	listenAddr   = "127.0.0.1:8081"
)

var (
	logLevel  string
	logFormat string
)

var rootCmd = &cobra.Command{
	Use:   "bitsongms",
	Short: "bitsongms implements a BitSong Media Server utility API.",
}

func init() {
	rootCmd.AddCommand(getStartCmd())
	rootCmd.AddCommand(getVersionCmd())
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func MakeCodec() *codec.Codec {
	var cdc = codec.New()

	sdk.RegisterCodec(cdc)
	types.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	codec.RegisterEvidences(cdc)

	return cdc
}

func getStartCmd() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start BitSong Media Server",
		RunE: func(cmd *cobra.Command, args []string) error {
			logLvl, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				return err
			}

			zerolog.SetGlobalLevel(logLvl)

			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				if err := os.Mkdir(dbPath, os.ModePerm); err != nil {
					return err
				}
			}

			// SDK Config
			config := sdk.GetConfig()
			config.SetBech32PrefixForAccount("bitsong", "bitsongpub")
			config.Seal()

			// Make Codec
			cdc := MakeCodec()

			// Start IPFS
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ipfsNode := ipfs.Start(ctx)

			// make a queue with a capacity of 1 transcoder.
			queue := make(chan *transcoder.Transcoder, 1)

			go func() {
				for q := range queue {
					doTranscode(q, ipfsNode)
				}
			}()

			// create HTTP router and mount routes
			router := mux.NewRouter()
			c := cors.New(cors.Options{
				AllowedOrigins: []string{"*"},
			})

			server.RegisterRoutes(router, queue, ipfsNode, cdc)

			srv := &http.Server{
				Handler:      c.Handler(router),
				Addr:         listenAddr,
				WriteTimeout: 15 * time.Second,
				ReadTimeout:  15 * time.Second,
			}

			log.Info().Str("address", listenAddr).Msg("starting API server...")
			return srv.ListenAndServe()
		},
	}

	startCmd.Flags().StringVar(&logLevel, "log-level", zerolog.InfoLevel.String(), "logging level")
	startCmd.Flags().StringVar(&logFormat, "log-format", logLevelJSON, "logging format; must be either json or text")

	return startCmd
}

func doTranscode(audio *transcoder.Transcoder, ipfsNode icore.CoreAPI) {
	tm := &models.Transcoder{
		ID: audio.Id,
	}

	tm.UpdatePercentage(20)
	// Convert to mp3
	log.Info().Str("filename", audio.Uploader.Header.Filename).Msg("starting conversion to mp3")

	if err := audio.TranscodeToMp3(); err != nil {
		log.Error().Str("filename", audio.Uploader.Header.Filename).Msg("failed to transcode")
		return
	}

	tm.UpdatePercentage(50)

	// check size compared to original

	// spilt mp3 to segments
	log.Info().Str("filename", audio.Uploader.Header.Filename).Msg("starting splitting to segments")

	if err := audio.SplitToSegments(); err != nil {
		log.Error().Str("filename", audio.Uploader.Header.Filename).Msg("failed to split")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get list of segments *.ts
	// For each segment upload to ipfs
	err := filepath.Walk(audio.Uploader.GetDir(), func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".ts") {
			fmt.Println(path)
			segment, err := utils.GetUnixfsNode(path)
			if err != nil {
				panic(fmt.Errorf("Could not get File: %s", err))
			}

			cidFile, err := ipfsNode.Unixfs().Add(ctx, segment)
			if err != nil {
				panic(fmt.Errorf("Could not add File: %s", err))
			}

			fmt.Println("Added file to IPFS with CID %s\n", cidFile.String())

			// Replace string into m3u8
			listFileName := audio.Uploader.GetDir() + "list.m3u8"
			list, err := ioutil.ReadFile(listFileName)
			if err != nil {
				panic(fmt.Errorf("Cannot read list: %s", err))
			}

			oldFileName := strings.Replace(path, audio.Uploader.GetDir(), "", -1)
			newFileName := fmt.Sprintf("%s", cidFile.String())

			listReplaced := bytes.Replace(list, []byte(oldFileName), []byte(newFileName), -1)

			// Change segment to hash in list.m3u8
			if err = ioutil.WriteFile(listFileName, listReplaced, 0666); err != nil {
				panic(fmt.Errorf("Cannot update list: %s", err))
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	// Upload list.m3u8 to ipfs
	listFile, err := utils.GetUnixfsNode(audio.Uploader.GetDir() + "list.m3u8")
	if err != nil {
		panic(fmt.Errorf("Could not get File: %s", err))
	}

	cidFile, err := ipfsNode.Unixfs().Add(ctx, listFile)
	if err != nil {
		panic(fmt.Errorf("Could not add File: %s", err))
	}

	fmt.Println(fmt.Sprintf("Added list.m3u8 to IPFS with CID %s\n", cidFile.String()))

	// Save cid list.m3u8 to transcoder collection
	tm.AddList(cidFile.String())

	// Upload original file to ipfs
	originaFile, err := utils.GetUnixfsNode(audio.Uploader.GetDir() + "original.mp3")
	if err != nil {
		panic(fmt.Errorf("Could not get File: %s", err))
	}

	cidFile, err = ipfsNode.Unixfs().Add(ctx, originaFile)
	if err != nil {
		panic(fmt.Errorf("Could not add File: %s", err))
	}

	fmt.Println(fmt.Sprintf("Added original.mp3 to IPFS with CID %s\n", cidFile.String()))

	// Save cid original file to transcoder collection
	tm.AddOriginal(cidFile.String())

	// remove all files
	audio.Uploader.RemoveAll()

	// TODO: Do not forget to pin everything

	tm.UpdatePercentage(100)

	log.Info().Str("filename", audio.Uploader.Header.Filename).Msg("transcode completed")
}
