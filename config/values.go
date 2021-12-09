package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/go-hclog"
)

type Values struct {
	// A unique ID for this server across all time.
	ServerId string
	// Advertise address to use.
	AdvertiseAddr string
	// A port raft gRPC server listens to
	GRPCPort int
	// A port raft gRPC-Gateway server listens to
	GRPCGWPort int // Path to a data directory to store raftkv state.
	DataDir    string
	// An address to send a join request.
	JoinAddr string
	// How many connections we will pool
	MaxPool int
	// In-memory storage for Raft.
	InMemory bool
	LogLevel int
}

// Load loads environment variables.
func (v *Values) Load() error {
	// Required values
	flag.StringVar(&v.ServerId, "server-id", os.Getenv("SERVER_ID"), "a unique ID for this server across all time")
	flag.StringVar(&v.AdvertiseAddr, "advertise-addr", os.Getenv("ADVERTISE_ADDR"), "Sets the advertise address to use")
	flag.IntVar(&v.GRPCPort, "grpc-port", getEnvInt("GRPC_PORT", 0), "a port raft gRPC server listens to")
	flag.IntVar(&v.GRPCGWPort, "grpcgw-port", getEnvInt("GRPC_GATEWAY_PORT", 0), "a port raft gRPC-Gateway server listens to")
	flag.StringVar(&v.DataDir, "data-dir", os.Getenv("DATA_DIR"), "path to a data directory to store raftkv state")

	// Optional values
	flag.StringVar(&v.JoinAddr, "join-addr", os.Getenv("JOIN_ADDR"), "an address to send a join request")
	flag.IntVar(&v.MaxPool, "maxpool", getEnvInt("GRPC_GATEWAY_PORT", 3), "how many connections we will pool")
	flag.BoolVar(&v.InMemory, "in-memory", false, "")
	flag.IntVar(&v.LogLevel, "log-level", getEnvInt("LOG_LEVEL", 1), "")

	flag.Parse()

	return v.validate()
}

// Validate validates configuration.
func (v *Values) validate() error {
	if v.ServerId == "" {
		return fmt.Errorf("server id is missing")
	}
	if v.AdvertiseAddr == "" {
		return fmt.Errorf("advertise addr is missing")
	}
	if v.GRPCPort < 0 {
		return fmt.Errorf("gRPC port is missing")
	}
	if v.GRPCGWPort < 0 {
		return fmt.Errorf("gRPC-Gateway port is missing")
	}
	if v.DataDir == "" {
		return fmt.Errorf("data dir is missing")
	}
	if v.MaxPool < 1 {
		v.MaxPool = 3
	}
	if v.LogLevel < 1 {
		v.LogLevel = int(hclog.DefaultLevel)
	}
	return nil
}

func (v *Values) GRPCAddr() string {
	return fmt.Sprintf(":%d", v.GRPCPort)
}

func (v *Values) GRPCGWAddr() string {
	return fmt.Sprintf(":%d", v.GRPCGWPort)
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
