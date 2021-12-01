package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/kei6u/raftkv"
	"github.com/kei6u/raftkv/fsm"
	raftkvpb "github.com/kei6u/raftkv/proto/v1"
	"google.golang.org/grpc"
)

func main() {
	var logLevel int
	if err := raftkv.LoadInt("LOG_LEVEL", &logLevel); err != nil {
		logLevel = int(hclog.Info)
	}
	hclog.DefaultOptions.Level = hclog.Level(logLevel)
	l := hclog.New(hclog.DefaultOptions)

	var (
		// required
		serverId   string
		raftAddr   string
		grpcAddr   string
		grpcgwAddr string
		// optional
		dir      string
		joinAddr string
		maxPool  int
		retain   int
		timeout  int
	)

	// Required values
	if err := raftkv.LoadString("SERVER_ID", &serverId); err != nil {
		l.Warn(err.Error())
		os.Exit(1)
	}
	if err := raftkv.LoadAddr("RAFT_ADDR", &raftAddr); err != nil {
		l.Warn(err.Error())
		os.Exit(1)
	}
	if err := raftkv.LoadAddr("GRPC_ADDR", &grpcAddr); err != nil {
		l.Warn(err.Error())
		os.Exit(1)
	}
	if err := raftkv.LoadAddr("GRPC_GATEWAY_ADDR", &grpcgwAddr); err != nil {
		l.Warn(err.Error())
		os.Exit(1)
	}

	// Optional values
	if err := raftkv.LoadString("FILE_SNAPSHOT_STORE_DIR", &dir); err != nil {
		l.Debug(err.Error())
		dir = fmt.Sprintf("_data/%s_%v.d", serverId, time.Now().Unix())
		l.Debug("set a file snapshot store directory forcefully", "dir", dir)
	}
	os.MkdirAll(dir, 0700)
	if err := raftkv.LoadString("JOIN_ADDR", &joinAddr); err != nil {
		l.Debug(err.Error())
	}
	if err := raftkv.LoadInt("MAX_POOL", &maxPool); err != nil {
		l.Debug(err.Error())
	}
	if err := raftkv.LoadInt("RETAIN", &retain); err != nil {
		l.Debug(err.Error())
	}
	if err := raftkv.LoadInt("TIMEOUT_SECOND", &timeout); err != nil {
		l.Debug(err.Error())
	}

	sig := make(chan os.Signal, 1)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	s := fsm.NewStore(
		dir,
		raftAddr,
		l,
		fsm.WithMaxPool(maxPool),
		fsm.WithRetain(retain),
		fsm.WithTimeoutSecond(timeout),
	)
	if joinAddr == "" {
		if err := s.OpenAsLeader(ctx, serverId); err != nil {
			l.Warn("exit due to a failure of opening a store as a leader", "error", err)
			os.Exit(1)
		}
	} else {
		if err := s.Open(ctx, serverId); err != nil {
			l.Warn("exit due to a failure of opening a store", "error", err)
			os.Exit(1)
		}
		conn, err := grpc.DialContext(ctx, joinAddr, grpc.WithInsecure())
		if err != nil {
			l.Warn("exit due to a failure of dialing a gRPC server to join", "error", err)
			cancel()
			return
		}
		if _, err := raftkvpb.NewRaftkvServiceClient(conn).Join(ctx, &raftkvpb.JoinRequest{
			NodeId:  serverId,
			Address: raftAddr,
		}); err != nil {
			l.Warn("exit due to a failure of joining", "error", err)
			cancel()
			return
		}
	}

	server, err := raftkvpb.NewServer(ctx, grpcAddr, grpcgwAddr, l, s)
	if err != nil {
		l.Warn("exit due to a failure of a server", "error", err)
		cancel()
	}
	if err := server.Start(ctx, grpcAddr); err != nil {
		l.Warn("exit due to a failure of starting a server", "error", err)
		cancel()
	}

	<-ctx.Done()
	server.Stop()
}
