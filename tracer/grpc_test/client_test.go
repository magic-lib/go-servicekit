package example_test

const (
	address = "localhost:50051"
)

//func TestGrpcTraceClient(t *testing.T) {
//	_, _ = tc.InitTrace()
//
//	_, clitOpt := tc.GrpcMiddleware()
//	opts := []grpc.DialOption{
//		clitOpt,
//		grpc.WithTransportCredentials(insecure.NewCredentials()),
//	}
//
//	conn, err := grpc.NewClient(address, opts...)
//	if err != nil {
//		log.Fatalf("did not connect: %v", err)
//	}
//	defer conn.Close()
//
//	c := pb.NewExampleServiceClient(conn)
//
//	ctx := context.Background()
//
//	ctx, span := tc.StartSpan(ctx, "SayHello-Client")
//
//	in := &pb.HelloRequest{Name: "tianlin"}
//	resp, err := c.SayHello(ctx, in)
//	if err != nil {
//		log.Fatalf("could not greet: %v", err)
//	}
//
//	tracer.SetTags(span, map[string]any{
//		"response": resp.Message,
//	})
//
//	span.End()
//
//	fmt.Printf("Greeting: %s\n", resp.Message)
//
//	time.Sleep(5 * time.Second)
//}
