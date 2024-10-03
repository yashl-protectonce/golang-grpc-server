package main

import (
    "context"
    "io"
    "net"
    "strconv"
    "testing"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
    proto "grpc/protoc"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
    lis = bufconn.Listen(bufSize)
    s := grpc.NewServer()
    proto.RegisterExampleServer(s, &server{})
    go func() {
        if err := s.Serve(lis); err != nil {
            panic(err)
        }
    }()
}

func bufDialer(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}

func TestServerReply(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()

    client := proto.NewExampleClient(conn)
    resp, err := client.ServerReply(ctx, &proto.HelloRequest{SomeString: "test"})
    if err != nil {
        t.Fatalf("ServerReply failed: %v", err)
    }

    if resp == nil {
        t.Error("Expected a response, got nil")
    }
}

func TestClientServerReply(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()

    client := proto.NewExampleClient(conn)
    stream, err := client.ClientServerReply(ctx)
    if err != nil {
        t.Fatalf("ClientServerReply failed: %v", err)
    }

    for i := 0; i < 3; i++ {
        if err := stream.Send(&proto.HelloRequest{SomeString: "test" + strconv.Itoa(i)}); err != nil {
            t.Fatalf("Failed to send request: %v", err)
        }
    }

    reply, err := stream.CloseAndRecv()
    if err != nil {
        t.Fatalf("Failed to receive reply: %v", err)
    }

    if reply.Reply != "3" {
        t.Errorf("Expected reply 3, got %v", reply.Reply)
    }
}

func TestServerStreamReply(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()

    client := proto.NewExampleClient(conn)
    stream, err := client.ServerStreamReply(ctx, &proto.HelloRequest{SomeString: "test"})
    if err != nil {
        t.Fatalf("ServerStreamReply failed: %v", err)
    }

    var responses []*proto.HelloResponse
    for {
        resp, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            t.Fatalf("Failed to receive response: %v", err)
        }
        responses = append(responses, resp)
    }

    expectedResponses := 4
    if len(responses) != expectedResponses {
        t.Errorf("Expected %d responses, got %d", expectedResponses, len(responses))
    }
}

func TestBilateralStreamReply(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()

    client := proto.NewExampleClient(conn)
    stream, err := client.BilateralStreamReply(ctx)
    if err != nil {
        t.Fatalf("BilateralStreamReply failed: %v", err)
    }

    done := make(chan bool)

    go func() {
        for i := 0; i < 5; i++ {
            resp, err := stream.Recv()
            if err == io.EOF {
                break
            }
            if err != nil {
                t.Fatalf("Failed to receive response: %v", err)
            }
            t.Logf("Received: %s", resp.Reply)
        }
        done <- true
    }()

    for i := 0; i < 5; i++ {
        if err := stream.Send(&proto.HelloRequest{SomeString: "test" + strconv.Itoa(i)}); err != nil {
            t.Fatalf("Failed to send request: %v", err)
        }
        time.Sleep(1 * time.Second)
    }
    err = stream.CloseSend()
    if err != nil {
        t.Fatalf("Failed to close stream send: %v", err)
    }

    <-done
}