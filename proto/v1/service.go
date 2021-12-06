package raftkvpb

import (
	"context"
	"errors"

	"github.com/kei6u/raftkv/fsm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

var _ RaftkvServiceServer = (*raftkvService)(nil)

type raftkvService struct {
	fsm *fsm.Server
}

func newRaftkvService(s *fsm.Server) *raftkvService { return &raftkvService{fsm: s} }

func (s *raftkvService) Get(_ context.Context, req *GetRequest) (*wrapperspb.StringValue, error) {
	v, err := s.fsm.Get(req.GetKey())
	if err != nil {
		if errors.Is(err, fsm.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return wrapperspb.String(v), nil
}

func (s *raftkvService) Set(_ context.Context, req *SetRequest) (*emptypb.Empty, error) {
	if err := s.fsm.Set(req.GetKey(), req.GetValue()); err != nil {
		if errors.Is(err, fsm.ErrEmptyKey) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *raftkvService) Delete(_ context.Context, req *DeleteRequest) (*emptypb.Empty, error) {
	if err := s.fsm.Delete(req.GetKey()); err != nil {
		if errors.Is(err, fsm.ErrEmptyKey) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *raftkvService) Join(_ context.Context, req *JoinRequest) (*emptypb.Empty, error) {
	if err := s.fsm.Join(req.GetServerId(), req.GetAddress()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}
