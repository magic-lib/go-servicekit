package example_test

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-servicekit/tracer"
	pb "github.com/magic-lib/go-servicekit/tracer/grpc_test/example"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"testing"
)

const (
	port = ":50051"
)

type server struct {
	pb.UnimplementedExampleServiceServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
	_, span := tc.StartSpan(ctx, "SayHello-Server")
	defer span.End()
	tracer.SetTags(span, map[string]any{
		"name": in.Name,
	})

	fmt.Printf("Received: %v\n", in.Name)
	return &pb.HelloResponse{Message: "Hello " + in.Name}, nil
}

func TestGrpcTraceServer(t *testing.T) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	_, _ = tc.InitTrace()

	serOpt, _ := tc.GrpcMiddleware()

	opts := []grpc.ServerOption{
		serOpt,
		grpc.Creds(insecure.NewCredentials()),
	}
	s := grpc.NewServer(opts...)
	pb.RegisterExampleServiceServer(s, &server{})

	fmt.Println("Server listening at", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
