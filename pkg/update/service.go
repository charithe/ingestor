package update

import (
	"io"

	"github.com/charithe/ingestor/pkg/v1pb"
	"go.uber.org/zap"
	"google.golang.org/grpc/health"
)

// Service implements the gRPC updater service
type Service struct {
	*health.Server
	db Database
}

// NewService creates a new instance of the service
func NewService(db Database) *Service {
	return &Service{
		Server: health.NewServer(),
		db:     db,
	}
}

// Update is the streaming RPC method exposed by the service
func (s *Service) Update(stream v1pb.Updater_UpdateServer) error {
	recordCount := 0

	for {
		rec, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				zap.S().Infow("Update finished", "recordCount", recordCount)
				return nil
			}

			zap.S().Errorw("Error while receiving message from the stream", "recordCount", recordCount, "error", err)
			return err
		}

		response := &v1pb.UpdateResponse{
			Id:     rec.Id,
			Status: v1pb.UpdateStatus_OK,
		}

		wrappedRec := Record(*rec)

		// TODO implement circuit breaker and retry logic
		if err := s.db.Upsert(stream.Context(), &wrappedRec); err != nil {
			zap.S().Warnw("Failed to update record", "record", rec, "error", err)
			response.Status = v1pb.UpdateStatus_ERROR
		}

		recordCount++

		if err := stream.Send(response); err != nil {
			zap.S().Errorw("Failed to send response back to client", "error", err)
			return err
		}
	}
}
