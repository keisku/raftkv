package raftkvpb

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func hclogUnaryInterceptor(l hclog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		res, err := handler(ctx, req)
		if err != nil {
			l.Error(fmt.Sprintf("%s is failed with %s", info.FullMethod, status.Code(err).String()), "error", err)
			return nil, err
		}
		l.Info(fmt.Sprintf("%s is finished with %s", info.FullMethod, status.Code(err).String()))
		return res, nil
	}
}
