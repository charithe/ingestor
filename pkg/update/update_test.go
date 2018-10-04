package update

import (
	"context"
	"net"
	"testing"

	"github.com/charithe/ingestor/pkg/v1pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestIntegration(t *testing.T) {
	db := NewInMemDB()
	svc := NewService(db)

	srv, addr := startServer(t, svc)
	defer srv.Stop()

	client := createClient(t, addr)
	defer client.Close()

	ctx := context.Background()
	sess, err := client.StartSession(ctx)
	if err != nil {
		t.Fatal(err)
	}

	defer sess.Close()

	assert.NoError(t, sess.Update(ctx, &Record{Id: 1, Name: "Alice", Email: "alice@example.com", MobileNumber: "08002345678"}))
	assert.NoError(t, sess.Update(ctx, &Record{Id: 2, Name: "Alice Smith", Email: "alice@example.com", MobileNumber: "08002348765"}))
	assert.NoError(t, sess.Update(ctx, &Record{Id: 3, Name: "Bob", Email: "bob@example.com", MobileNumber: "07878787878"}))
	assert.NoError(t, sess.Update(ctx, &Record{Id: 4, Name: "Bobby", Email: "bob@example.com", MobileNumber: "07989898989"}))

	results, _ := db.List(ctx)
	assert.Len(t, results, 2)

	assert.Equal(t, "Alice Smith", results[0].Name)
	assert.Equal(t, "Bobby", results[1].Name)

}

func startServer(t *testing.T, svc *Service) (*grpc.Server, string) {
	t.Helper()

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	addr := lis.Addr().String()
	srv := grpc.NewServer()
	v1pb.RegisterUpdaterServer(srv, svc)

	go func() {
		srv.Serve(lis)
	}()

	return srv, addr
}

func createClient(t *testing.T, addr string) *Client {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}

	return NewClient(conn)
}
