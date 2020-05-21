package cmd

import (
	"fmt"
	"github.com/bitsongofficial/bstudio/ds"
	"github.com/bitsongofficial/bstudio/server"
	"github.com/bitsongofficial/bstudio/transcoder"
	"github.com/gorilla/mux"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"time"
)

const (
	logLevelJSON = "json"
	logLevelText = "text"
	listenAddr   = "127.0.0.1:1347"
)

var (
	logLevel  string
	logFormat string
	ipfsAddr  string
)

var rootCmd = &cobra.Command{
	Use:   "bstudio",
	Short: "bstudio implements a BitSong Upload and Transcoding utility API.",
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

func getStartCmd() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start BitSong Studio API",
		RunE: func(cmd *cobra.Command, args []string) error {
			logLvl, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				return err
			}

			zerolog.SetGlobalLevel(logLvl)

			if _, err := os.Stat(".bstudio"); os.IsNotExist(err) {
				if err := os.Mkdir(".bstudio", os.ModePerm); err != nil {
					return err
				}
			}

			// Start IPFS Shell
			sh := shell.NewShell(ipfsAddr)
			if !sh.IsUp() {
				return fmt.Errorf("ipfs api is down!")
			}

			// Create datastore
			ds := ds.NewDs()
			defer ds.Db.Close()

			// Create context
			//ctx, cancel := context.WithCancel(context.Background())
			//defer cancel()

			// make a queue with a capacity of 1 transcoder.
			queue := make(chan *transcoder.Transcoder, 1)

			go func() {
				for q := range queue {
					doTranscode(q, sh)
				}
			}()

			// create HTTP router and mount routes
			router := mux.NewRouter()
			c := cors.New(cors.Options{
				AllowedOrigins: []string{"*"},
				//AllowedMethods: []string{"GET", "PUT", "PATCH", "POST", "DELETE", "OPTIONS"},
				//AllowedHeaders: []string{"Origin", "Authorization", "Content-Type", "Accept", "Access-Control-Allow-Methods", "Access-Control-Request-Headers"},
				//ExposedHeaders: []string{""},
				//MaxAge: 10,
				//AllowCredentials: true,
			})

			server.RegisterRoutes(router, queue, sh, ds)

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
	startCmd.Flags().StringVar(&ipfsAddr, "ipfs-addr", "localhost:5001", "ipfs api address")

	return startCmd
}

func doTranscode(audio *transcoder.Transcoder, sh *shell.Shell) {
	fmt.Println("starting transcoding " + audio.Uploader.ID.String())

	if err := audio.UpdatePercentage(20); err != nil {
		panic(err)
	}

	// Convert to mp3
	log.Info().Str("filename", audio.Uploader.Header.Filename).Msg("starting conversion to mp3")

	if err := audio.TranscodeToMp3(); err != nil {
		log.Error().Str("filename", audio.Uploader.Header.Filename).Msg("failed to transcode")
		return
	}

	if err := audio.UpdatePercentage(50); err != nil {
		panic(err)
	}

	// check size compared to original

	// spilt mp3 to segments
	log.Info().Str("filename", audio.Uploader.Header.Filename).Msg("starting splitting to segments")

	if err := audio.SplitToSegments(); err != nil {
		log.Error().Str("filename", audio.Uploader.Header.Filename).Msg("failed to split")
		return
	}

	// Get list of segments *.ts
	// For each segment upload to ipfs
	log.Info().Str("filename", audio.Uploader.Header.Filename).Msg("uploading to ipfs")
	cid, err := sh.AddDir(audio.Uploader.GetDir())
	if err != nil {
		panic(err)
	}
	log.Info().Str("filename", audio.Uploader.Header.Filename).Msg("has been uploaded " + cid)
	/*err := filepath.Walk(audio.Uploader.GetDir(), func(path string, info os.FileInfo, err error) error {
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

			fmt.Println(fmt.Sprintf("Added file to IPFS with CID %s\n", cidFile.String()))

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

	// update track db
	track, err := models.GetTrack(audio.TrackID)
	if err != nil {
		panic(fmt.Errorf("Could not update track: %s", err))
	}

	track.Audio = cidFile.String()
	err = track.Update()
	if err != nil {
		panic(fmt.Errorf("Could not update track: %s", err))
	}

	// Upload original file to ipfs
	err = filepath.Walk(audio.Uploader.GetDir(), func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, "original.") {
			originalFile, err := utils.GetUnixfsNode(path)
			if err != nil {
				panic(fmt.Errorf("Could not get File: %s", err))
			}

			cidFile, err = ipfsNode.Unixfs().Add(ctx, originalFile)
			if err != nil {
				panic(fmt.Errorf("Could not add File: %s", err))
			}

			track.AudioOriginal = cidFile.String()
			err = track.Update()
			if err != nil {
				panic(fmt.Errorf("Could not update track: %s", err))
			}

			fmt.Printf("Added original file to IPFS with CID %s\n", cidFile.String())

			// Save cid original file to transcoder collection
			tm.AddOriginal(cidFile.String())

			return io.EOF

		}
		return nil
	})

	if err == io.EOF {
		err = nil
	}

	if err != nil {
		panic(err)
	}*/

	// remove all files
	//audio.Uploader.RemoveAll()

	// TODO: Do not forget to pin everything

	if err := audio.UpdatePercentage(100); err != nil {
		panic(err)
	}

	log.Info().Str("filename", audio.Uploader.Header.Filename).Msg("transcode completed")
}
