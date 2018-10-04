package ingest

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/charithe/ingestor/pkg/update"
	"go.uber.org/zap"
)

var nonNumericRegex = regexp.MustCompile("[^0-9]+")

// Ingestor implements methods to parse a CSV stream and send update requests to the database
type Ingestor struct {
	stream  io.Reader
	updater update.Updater
}

// New creates an instance of an Ingestor bound to the given stream
func New(stream io.Reader, updater update.Updater) *Ingestor {
	return &Ingestor{
		stream:  stream,
		updater: updater,
	}
}

// Start kicks off the ingestion process
func (ing *Ingestor) Start(ctx context.Context) error {
	// check whether the context is already cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	csvReader := csv.NewReader(ing.stream)
	csvReader.ReuseRecord = true

	// read header
	if _, err := csvReader.Read(); err != nil {
		zap.S().Errorw("Failed to read header from the stream", "error", err)
		return err
	}

	for {
		select {
		case <-ctx.Done():
			zap.S().Warnw("Context cancelled before reaching end of stream", "error", ctx.Err())
			return ctx.Err()

		default:
			rec, err := parseNextRecord(csvReader)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				zap.S().Errorw("Failed to parse record from the stream", "error", err)
				return err
			}

			if err := ing.updater.Update(ctx, rec); err != nil {
				zap.S().Errorw("Failed to update record", "record", rec, "error", err)
				if err != update.ErrRecordNotUpdated {
					return err
				}
			}
		}
	}
}

func parseNextRecord(r *csv.Reader) (*update.Record, error) {
	tmpRec, err := r.Read()
	if err != nil {
		return nil, err
	}

	if len(tmpRec) != 4 {
		return nil, fmt.Errorf("invalid record")
	}

	id, err := strconv.ParseInt(tmpRec[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID value [%s]: %+v", tmpRec[0], err)
	}

	return &update.Record{
		Id:           id,
		Name:         tmpRec[1],
		Email:        tmpRec[2],
		MobileNumber: formatMobileNumber(tmpRec[3]),
	}, nil
}

func formatMobileNumber(value string) string {
	cleaned := nonNumericRegex.ReplaceAllString(value, "")
	if strings.HasPrefix(cleaned, "44") {
		cleaned = strings.TrimPrefix(cleaned, "44")
	} else if strings.HasPrefix(cleaned, "0044") {
		cleaned = strings.TrimPrefix(cleaned, "0044")
	}

	return cleaned
}
