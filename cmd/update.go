package cmd

import (
	"net"
	"os"
	"os/signal"

	"github.com/charithe/ingestor/pkg/update"
	"github.com/charithe/ingestor/pkg/v1pb"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type updateArgs struct {
	listenAddr string
}

var updateCmdArgs = updateArgs{}

func createUpdateCommand() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Start update service",
		RunE:  doUpdate,
	}

	updateCmd.Flags().StringVar(&updateCmdArgs.listenAddr, "listen", ":8090", "Listen address")
	return updateCmd
}

func doUpdate(_ *cobra.Command, _ []string) error {
	lis, err := net.Listen("tcp", updateCmdArgs.listenAddr)
	if err != nil {
		zap.S().Errorw("Failed to create listener", "error", err)
		return err
	}
	defer lis.Close()

	db := update.NewInMemDB()
	updaterSvc := update.NewService(db)

	grpcLogger := zap.L().Named("grpc")

	codeToLevel := grpc_zap.CodeToLevel(func(code codes.Code) zapcore.Level {
		if code == codes.OK {
			return zapcore.DebugLevel
		}
		return grpc_zap.DefaultCodeToLevel(code)
	})

	serverOpts := []grpc.ServerOption{
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(grpcLogger, grpc_zap.WithLevels(codeToLevel)),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_zap.StreamServerInterceptor(grpcLogger, grpc_zap.WithLevels(codeToLevel)),
		),
	}

	grpcServer := grpc.NewServer(serverOpts...)
	v1pb.RegisterUpdaterServer(grpcServer, updaterSvc)
	healthpb.RegisterHealthServer(grpcServer, updaterSvc)

	reflection.Register(grpcServer)

	go func() {
		zap.S().Info("Starting update server")
		if err := grpcServer.Serve(lis); err != nil {
			zap.S().Fatalw("Update server failed", "error", err)
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt)
	<-shutdownChan

	zap.S().Info("Shutting down")
	grpcServer.GracefulStop()

	return nil
}
