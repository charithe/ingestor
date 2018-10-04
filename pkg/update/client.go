package update

import (
	"context"
	"fmt"
	"sync"

	"github.com/charithe/ingestor/pkg/v1pb"
	"google.golang.org/grpc"
)

var (
	// ErrSessionClosed is returned when the Session is no longer valid
	ErrSessionClosed = fmt.Errorf("session closed")
	// ErrRecordNotUpdated is returned if the record could not be updated. This is a transient error
	ErrRecordNotUpdated = fmt.Errorf("record not updated")
)

// Client is an RPC client for the Updater service
type Client struct {
	conn   *grpc.ClientConn
	client v1pb.UpdaterClient
}

// NewClient creates a new instance of client
func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{
		conn:   conn,
		client: v1pb.NewUpdaterClient(conn),
	}
}

// StartSession begins a new update session with the remote server
func (c *Client) StartSession(ctx context.Context) (*Session, error) {
	streamClient, err := c.client.Update(ctx)
	if err != nil {
		return nil, err
	}

	return &Session{
		closed:       make(chan struct{}),
		streamClient: streamClient,
	}, nil
}

// Close closed the client connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// Session represents an ongoing update session with the remote server
type Session struct {
	closeSession sync.Once
	closed       chan struct{}
	streamClient v1pb.Updater_UpdateClient
}

// Update implements the Update method from the Updater interface
func (s *Session) Update(ctx context.Context, rec *Record) error {
	select {
	case <-s.closed:
		return ErrSessionClosed
	case <-ctx.Done():
		s.Close()
		return ctx.Err()
	default:
	}

	// TODO properly implement context cancellation

	wrappedRec := v1pb.UpdateRequest(*rec)
	if err := s.streamClient.Send(&wrappedRec); err != nil {
		s.Close()
		return err
	}

	resp, err := s.streamClient.Recv()
	if err != nil {
		s.Close()
		return err
	}

	if resp.Status == v1pb.UpdateStatus_ERROR {
		return ErrRecordNotUpdated
	}

	return nil
}

// Close closes the session
func (s *Session) Close() {
	s.closeSession.Do(func() {
		select {
		case <-s.closed:
		default:
			close(s.closed)
			s.streamClient.CloseSend()
		}
	})
}
