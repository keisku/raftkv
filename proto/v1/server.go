package raftkvpb

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hashicorp/go-hclog"
	"github.com/kei6u/raftkv/kv"
	"google.golang.org/grpc"
)

type Server struct {
	logger         hclog.Logger
	gRPCAddr       string
	gRPCServer     *grpc.Server
	gRPCListener   net.Listener
	gRPCClientConn *grpc.ClientConn
	gRPCGWServer   *http.Server
}

func NewServer(ctx context.Context, gRPCAddr, gRPCGWAddr string, l hclog.Logger, kvserver *kv.Server) (*Server, error) {
	grpcServer := newgRPCServer(kvserver, l)
	httpServer, conn, err := newgRPCGWServer(ctx, gRPCAddr, gRPCGWAddr)
	if err != nil {
		return nil, err
	}
	return &Server{
		logger:         l,
		gRPCAddr:       gRPCAddr,
		gRPCServer:     grpcServer,
		gRPCClientConn: conn,
		gRPCGWServer:   httpServer,
	}, nil
}

func newgRPCServer(server *kv.Server, l hclog.Logger) *grpc.Server {
	grpcserver := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			hclogUnaryInterceptor(l),
		),
	)
	RegisterRaftkvServiceServer(grpcserver, newRaftkvService(server))
	return grpcserver
}

func newgRPCGWServer(ctx context.Context, gRPCAddr, gRPCGWAddr string) (*http.Server, *grpc.ClientConn, error) {
	conn, err := grpc.DialContext(
		ctx,
		gRPCAddr,
		grpc.WithInsecure(),
		grpc.WithDisableHealthCheck(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial a gRPC server: %w", err)
	}

	mux := runtime.NewServeMux()
	if err := RegisterRaftkvServiceHandler(ctx, mux, conn); err != nil {
		return nil, nil, fmt.Errorf("failed to register a service handler: %w", err)
	}
	return &http.Server{
		Addr:    gRPCGWAddr,
		Handler: mux,
	}, conn, nil
}

func (s *Server) Start() error {
	s.logger.Info("Starting gRPC and HTTP servers")
	lis, err := net.Listen("tcp", s.gRPCAddr)
	if err != nil {
		return fmt.Errorf("failed to listen to gRPC addr: %w", err)
	}
	s.gRPCListener = lis

	go s.gRPCServer.Serve(lis)
	go s.gRPCGWServer.ListenAndServe()
	return nil
}

func (s *Server) Stop() {
	ctx := context.Background()
	if s.gRPCGWServer != nil {
		s.logger.Info("gRPC-Gateway server is shutting down")
		if err := s.gRPCGWServer.Shutdown(ctx); err != nil {
			s.logger.Error("failed to shutdown gRPC gateway server", "error", err)
		}
	}
	if err := s.gRPCListener.Close(); err != nil {
		s.logger.Error("failed to close listener to gRPC server", "error", err)
	}
	if s.gRPCServer != nil {
		s.logger.Info("gRPC server is gracefully stopping")
		s.gRPCServer.GracefulStop()
	}
	s.logger.Info("Bye~~")
}
