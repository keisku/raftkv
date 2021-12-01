package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/kei6u/raftkv/fsm"
	raftkvpb "github.com/kei6u/raftkv/proto/v1"
	"google.golang.org/grpc"
)

var (
	l hclog.Logger

	// required
	serverId   string
	raftAddr   string
	grpcAddr   string
	grpcgwAddr string

	// options
	dir      string
	joinAddr string
	loglevel int
	maxPool  int
	retain   int
	timeout  int
)

func init() {
	// Required values
	flag.StringVar(&serverId, "server-id", os.Getenv("SERVER_ID"), "a unique ID for this server across all time")
	flag.StringVar(&raftAddr, "raft-addr", os.Getenv("RAFT_ADDR"), "an address raft binds")
	flag.StringVar(&grpcAddr, "grpc-addr", os.Getenv("GRPC_ADDR"), "an address raft gRPC server listens to")
	flag.StringVar(&grpcgwAddr, "grpcgw-addr", os.Getenv("GRPC_GATEWAY_ADDR"), "an address raft gRPC-Gateway server listens to")

	// Optional values
	flag.StringVar(&dir, "dir", os.Getenv("SNAPSHOT_STORE_DIR"), "a directory for a snapshot store")
	flag.StringVar(&joinAddr, "join-addr", os.Getenv("JOIN_ADDR"), "an address to send a join request")
	flag.IntVar(&maxPool, "maxpool", getEnvInt("MAXPOOL", 3), "how many connections we will pool")
	flag.IntVar(&retain, "retain", getEnvInt("RETAIN", 2), "how many snapshots are retained")
	flag.IntVar(&timeout, "timeout", getEnvInt("TIMEOUT_SECOND", 10), "the amount of time we wait for the command to be started")
	flag.IntVar(&loglevel, "log-level", getEnvInt("LOG_LEVEL", 3), "")

	flag.Parse()

	// setup a logger
	hclog.DefaultOptions.Level = hclog.Level(loglevel)
	l = hclog.New(hclog.DefaultOptions)

	for k, v := range map[string]string{
		"server id":         serverId,
		"raft addr":         raftAddr,
		"grpc addr":         grpcAddr,
		"grpc gateway addr": grpcgwAddr,
	} {
		if v == "" {
			l.Warn(fmt.Sprintf("%s is required", k))
			os.Exit(1)
		}
	}

	dir = filepath.Join(dir, fmt.Sprintf("%s_%v.d", serverId, time.Now().Unix()))
	if err := os.MkdirAll(dir, 0700); err != nil {
		l.Warn("exit due to a failure of making a directory for a file snapshot store", "error", err)
		os.Exit(1)
	}
}

func main() {
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
			l.Warn("exit since opening a store as a leader fails", "error", err)
			os.Exit(1)
		}
	} else {
		if err := s.Open(ctx, serverId); err != nil {
			l.Warn("exit since opening a store fails", "error", err)
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
		_ = conn.Close()
	}

	server, err := raftkvpb.NewServer(ctx, grpcAddr, grpcgwAddr, l, s)
	if err != nil {
		l.Warn("exit due to a failure of initializing a server", "error", err)
		cancel()
	}
	if err := server.Start(ctx, grpcAddr); err != nil {
		l.Warn("exit due to a failure of starting a server", "error", err)
		cancel()
	}

	<-ctx.Done()
	server.Stop()
}

func getEnvInt(key string, defaulti int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaulti
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaulti
	}
	return i
}
