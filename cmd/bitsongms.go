package cmd

import (
	"fmt"
	"github.com/bitsongofficial/bitsong-media-server/models"
	"github.com/bitsongofficial/bitsong-media-server/transcoder"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"net/http"
	"os"
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

			// make a queue with a capacity of 1 transcoder.
			queue := make(chan *transcoder.Transcoder, 1)

			go func() {
				for q := range queue {
					doTranscode(q)
				}
			}()

			// create HTTP router and mount routes
			router := mux.NewRouter()
			c := cors.New(cors.Options{
				AllowedOrigins: []string{"*"},
			})

			server.RegisterRoutes(router, queue)

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

func doTranscode(audio *transcoder.Transcoder) {
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

	tm.UpdatePercentage(100)

	// remove all files
	audio.Uploader.RemoveAll()

	log.Info().Str("filename", audio.Uploader.Header.Filename).Msg("transcode completed")
}
