package cmd

import (
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/charithe/ingestor/pkg/ingest"
	"github.com/charithe/ingestor/pkg/update"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type ingestArgs struct {
	listenAddr         string
	updateSvcAddr      string
	requestTimeout     time.Duration
	maxUploadSizeBytes int64
}

var ingestCmdArgs = ingestArgs{}

func createIngestCommand() *cobra.Command {
	ingestCmd := &cobra.Command{
		Use:   "ingest",
		Short: "Start ingestion service",
		RunE:  doIngest,
	}

	ingestCmd.Flags().StringVar(&ingestCmdArgs.listenAddr, "listen", ":8080", "Listen address")
	ingestCmd.Flags().StringVar(&ingestCmdArgs.updateSvcAddr, "update_svc", "localhost:8090", "Address of the update service")
	ingestCmd.Flags().DurationVar(&ingestCmdArgs.requestTimeout, "timeout", 5*time.Minute, "Request timeout")
	ingestCmd.Flags().Int64Var(&ingestCmdArgs.maxUploadSizeBytes, "max_upload_bytes", 1*1024*1024*1024, "Maximum upload size in bytes")

	return ingestCmd
}

func doIngest(_ *cobra.Command, _ []string) error {
	lis, err := net.Listen("tcp", ingestCmdArgs.listenAddr)
	if err != nil {
		zap.S().Errorw("Failed to create listener", "error", err)
		return err
	}

	defer lis.Close()

	conn, err := grpc.Dial(ingestCmdArgs.updateSvcAddr, grpc.WithInsecure())
	if err != nil {
		zap.S().Errorw("Failed to dial update service", "error", err)
		return err
	}

	updateClient := update.NewClient(conn)
	defer updateClient.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r.Body != nil {
				io.Copy(ioutil.Discard, r.Body)
				r.Body.Close()
			}
		}()

		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			http.Error(w, "only POST or PUT allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := ingestData(r.Body, updateClient); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			io.Copy(ioutil.Discard, r.Body)
			r.Body.Close()
		}

		w.WriteHeader(http.StatusOK)
	})

	logger := zap.L().Named("http")
	httpServer := &http.Server{
		Handler:           mux,
		ErrorLog:          zap.NewStdLog(logger),
		ReadHeaderTimeout: ingestCmdArgs.requestTimeout,
		WriteTimeout:      ingestCmdArgs.requestTimeout,
		IdleTimeout:       ingestCmdArgs.requestTimeout,
	}

	go func() {
		zap.S().Infow("Starting ingest server")
		if err := httpServer.Serve(lis); err != nil && err != http.ErrServerClosed {
			zap.S().Fatalw("Failed to start ingest server", "error", err)
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt)
	<-shutdownChan

	zap.S().Info("Shutting down")
	ctx, cancelFunc := context.WithTimeout(context.Background(), ingestCmdArgs.requestTimeout)
	defer cancelFunc()
	httpServer.Shutdown(ctx)

	return nil
}

func ingestData(r io.Reader, updateClient *update.Client) error {
	f, err := ioutil.TempFile("", "ingest")
	if err != nil {
		zap.S().Errorw("Failed to create temporary file", "error", err)
		return err
	}

	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	var buf [1024]byte
	if _, err = io.CopyBuffer(f, io.LimitReader(r, ingestCmdArgs.maxUploadSizeBytes), buf[:]); err != nil {
		zap.S().Errorw("Failed to write data to disk", "error", err)
		return err
	}

	if err = f.Sync(); err != nil {
		zap.S().Errorw("Failed to sync data to disk", "error", err)
		return err
	}

	if _, err = f.Seek(0, 0); err != nil {
		zap.S().Errorw("Failed to seek to beginning of data file", "error", err)
		return err
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), ingestCmdArgs.requestTimeout)
	defer cancelFunc()

	sess, err := updateClient.StartSession(ctx)
	if err != nil {
		zap.S().Errorw("Failed to start session", "error", err)
		return err
	}
	defer sess.Close()

	ingestor := ingest.New(f, sess)
	if err = ingestor.Start(ctx); err != nil {
		zap.S().Errorw("Failed to ingest data", "error", err)
		return err
	}

	return nil
}
