package main

import (
    "context"
    "io"
    "log"
    "net"
	"fmt"
    "testing"

    pb "github.com/pramonow/go-grpc-server-streaming-example/src/proto"
    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
    lis = bufconn.Listen(bufSize)
    s := grpc.NewServer()
    pb.RegisterStreamServiceServer(s, &server{})
    go func() {
        if err := s.Serve(lis); err != nil {
            log.Fatalf("Server exited with error: %v", err)
        }
    }()
}

func bufDialer(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}

func TestFetchResponse(t *testing.T) {
    conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()

    client := pb.NewStreamServiceClient(conn)
    req := &pb.Request{Id: 1}
    stream, err := client.FetchResponse(context.Background(), req)
    if err != nil {
        t.Fatalf("FetchResponse failed: %v", err)
    }

    var responses []*pb.Response
    for {
        resp, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            t.Fatalf("Failed to receive a response: %v", err)
        }
        responses = append(responses, resp)
    }

    if len(responses) != 5 {
        t.Errorf("Expected 5 responses, got %d", len(responses))
    }

    for i, resp := range responses {
        expected := fmt.Sprintf("Request #%d For Id:1", i)
        if resp.Result != expected {
            t.Errorf("Expected response %q, got %q", expected, resp.Result)
        }
    }
}