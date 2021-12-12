package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-hclog"
	"github.com/kei6u/raftkv/config"
	"github.com/kei6u/raftkv/kv"
	raftkvpb "github.com/kei6u/raftkv/proto/v1"
	"google.golang.org/grpc"
)

var (
	l    hclog.Logger
	conf = &config.Values{}
)

func init() {
	// setup a logger
	l = hclog.New(hclog.DefaultOptions)

	if err := conf.Load(); err != nil {
		l.Warn("exit since loading environment variables failed", "error", err)
		os.Exit(1)
	}

	l.SetLevel(hclog.Level(conf.LogLevel))

	if err := os.MkdirAll(conf.DataDir, 0700); err != nil {
		l.Warn("exit since making a directory for a file snapshot store failed", "error", err)
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

	store := kv.NewStore(l)
	kvserver := kv.NewServer(store, l, conf)

	if err := kvserver.Start(); err != nil {
		l.Warn("exit since a store failed to be opened", "error", err)
		os.Exit(1)
	}

	if conf.JoinAddr == "" {
		l.Info("bootstraping a new cluster")
		if err := kvserver.BootstrapCluster(); err != nil {
			l.Warn("exit since bootstraping a cluster failed", "error", err)
			os.Exit(1)
		}
	} else {
		conn, err := grpc.DialContext(ctx, conf.JoinAddr, grpc.WithInsecure())
		if err != nil {
			l.Warn(fmt.Sprintf("failed to dial a gRPC server at %s", conf.JoinAddr), "error", err)
			cancel()
			return
		}
		c := raftkvpb.NewRaftkvServiceClient(conn)
		op := func() error {
			_, err = c.Join(ctx, &raftkvpb.JoinRequest{
				ServerId: conf.ServerId,
				Address:  conf.AdvertiseAddr,
			})
			return err
		}
		l.Info("new server is joining to a cluster", "server_id", conf.ServerId, "advertise_addr", conf.AdvertiseAddr)
		if err := backoff.Retry(op, backoff.NewExponentialBackOff()); err != nil {
			l.Warn("new server failed to join a cluster", "error", err)
			cancel()
			return
		}
		_ = conn.Close()
	}

	server, err := raftkvpb.NewServer(ctx, conf.GRPCAddr(), conf.GRPCGWAddr(), l, kvserver)
	if err != nil {
		l.Warn("exit due to a failure of initializing a server", "error", err)
		cancel()
	}
	if err := server.Start(conf.GRPCAddr()); err != nil {
		l.Warn("exit due to a failure of starting a server", "error", err)
		cancel()
	}

	<-ctx.Done()
	_ = kvserver.Shutdown()
	server.Stop()
}
