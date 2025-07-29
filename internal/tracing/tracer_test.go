package tracing

import (
	"context"
	"net"
	"sync"
	"testing"

	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
)

type otlpServer struct {
	collectortrace.UnimplementedTraceServiceServer
	mu       sync.Mutex
	requests []*collectortrace.ExportTraceServiceRequest
}

func (s *otlpServer) Export(ctx context.Context, req *collectortrace.ExportTraceServiceRequest) (*collectortrace.ExportTraceServiceResponse, error) {
	s.mu.Lock()
	s.requests = append(s.requests, req)
	s.mu.Unlock()
	return &collectortrace.ExportTraceServiceResponse{}, nil
}

func TestInit(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	otlp := &otlpServer{}
	collectortrace.RegisterTraceServiceServer(server, otlp)
	go server.Serve(lis)
	defer server.Stop()

	tracer, shutdown := Init(context.Background(), "svc", "mc", lis.Addr().String())
	ctx, span := tracer.Start(context.Background(), "test")
	span.End()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	otlp.mu.Lock()
	received := len(otlp.requests)
	otlp.mu.Unlock()
	if received == 0 {
		t.Errorf("no spans received")
	}
}
