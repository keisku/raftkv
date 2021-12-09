package raftkvpb

import (
	"context"
	"errors"

	"github.com/kei6u/raftkv/kv"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

var _ RaftkvServiceServer = (*raftkvService)(nil)

type raftkvService struct {
	kvServer *kv.Server
}

func newRaftkvService(kvs *kv.Server) *raftkvService { return &raftkvService{kvServer: kvs} }

func (s *raftkvService) Get(_ context.Context, req *GetRequest) (*wrapperspb.StringValue, error) {
	v, err := s.kvServer.ApplyGetOp(req.GetKey())
	if err != nil {
		if errors.Is(err, kv.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return wrapperspb.String(v), nil
}

func (s *raftkvService) Set(_ context.Context, req *SetRequest) (*emptypb.Empty, error) {
	if err := s.kvServer.ApplySetOp(req.GetKey(), req.GetValue()); err != nil {
		if errors.Is(err, kv.ErrEmptyKey) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *raftkvService) Delete(_ context.Context, req *DeleteRequest) (*emptypb.Empty, error) {
	if err := s.kvServer.ApplyDeleteOp(req.GetKey()); err != nil {
		if errors.Is(err, kv.ErrEmptyKey) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *raftkvService) Join(_ context.Context, req *JoinRequest) (*emptypb.Empty, error) {
	if err := s.kvServer.Join(req.GetServerId(), req.GetAddress()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}
