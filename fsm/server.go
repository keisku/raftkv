package fsm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

// Server holds a key-value store and behaves as a FSM.
type Server struct {
	serverId      raft.ServerID
	dataDir       string
	advertise     string
	kvstore       kvstore
	mu            sync.Mutex
	raft          *raft.Raft
	raftTransport *raft.NetworkTransport
	raftStore     *raftboltdb.BoltStore
	logger        hclog.Logger
	options       *Options
}

// NewServer initializes a server.
func NewServer(serverId, dataDir, advertise string, l hclog.Logger, opt ...Option) *Server {
	opts := newOptions(opt...)
	return &Server{
		serverId:  raft.ServerID(serverId),
		dataDir:   dataDir,
		advertise: advertise,
		kvstore:   make(kvstore),
		logger:    l,
		options:   opts,
	}
}

// Run runs a finite-state machine.
func (s *Server) Run() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", s.advertise)
	if err != nil {
		return fmt.Errorf("failed to resolve a TCP address: %w", err)
	}

	s.raftTransport, err = raft.NewTCPTransport(
		s.advertise,
		tcpAddr,
		s.options.maxPool,
		s.options.timeout,
		os.Stderr,
	)
	if err != nil {
		return fmt.Errorf("failed to build a new TCP transport: %w", err)
	}

	ss, err := raft.NewFileSnapshotStore(s.dataDir, s.options.retain, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to build a new TCP transport: %w", err)
	}

	var stableStore raft.StableStore
	var logStore raft.LogStore
	if s.options.useInMemoryStore {
		s := raft.NewInmemStore()
		stableStore = s
		logStore = s
	} else {
		boltdb, err := raftboltdb.NewBoltStore(filepath.Join(s.dataDir, "raft.db"))
		if err != nil {
			return fmt.Errorf("failed to create a stable store: %w", err)
		}
		stableStore, s.raftStore = boltdb, boltdb
		logStore, err = raft.NewLogCache(512, boltdb)
		if err != nil {
			return fmt.Errorf("failed to create a log store: %w", err)
		}
	}

	config := raft.DefaultConfig()
	config.LocalID = s.serverId
	config.Logger = s.logger

	s.raft, err = raft.NewRaft(config, s, logStore, stableStore, ss, s.raftTransport)
	if err != nil {
		return fmt.Errorf("failed to construct a new Raft server: %w", err)
	}

	return nil
}

// BootstrapCluster bootstrap a new cluster.
// There are no existing peers, then this server becomes the first server,
// and therefore leader of the cluster.
func (s *Server) BootstrapCluster() error {
	n, err := s.numVoters()
	if err != nil {
		return fmt.Errorf("failed to get the number of peers: %w", err)
	}
	if 1 < n {
		return fmt.Errorf("there are %d peers, cluster may be already constructed", n)
	}
	if err := s.raft.BootstrapCluster(raft.Configuration{Servers: []raft.Server{{
		ID:      s.serverId,
		Address: s.raftTransport.LocalAddr(),
	}}}).Error(); err != nil {
		return fmt.Errorf("failed to bootstrap a cluster: %w", err)
	}
	return nil
}

// Leave is used to prepare for a graceful shutdown.
func (s *Server) Leave() error {
	s.logger.Info("server starting leave")

	peersN, err := s.numVoters()
	if err != nil {
		return fmt.Errorf("failed to get the number of peers: %w", err)
	}

	isLeader := s.isLeader()
	if isLeader && 1 < peersN {
		err := s.raft.LeadershipTransfer().Error()
		if err == nil {
			isLeader = false
		} else {
			s.logger.Error("failed to transfer leadership, removing the server", "error", err)
			if err := s.raft.RemoveServer(s.serverId, 0, 0).Error(); err != nil {
				s.logger.Error("failed to remove ourself as raft peer", "error", err)
			}
		}
	}

	if !isLeader {
		left := false
		limit := time.Now().Add(5 * time.Second)
		for !left && time.Now().Before(limit) {
			time.Sleep(50 * time.Millisecond)

			future := s.raft.GetConfiguration()
			if err := future.Error(); err != nil {
				s.logger.Error("failed to get raft configuration", "error", err)
				break
			}

			left = true
			for _, server := range future.Configuration().Servers {
				if server.Address == s.raftTransport.LocalAddr() {
					left = false
					break
				}
			}
		}
		if !left {
			s.logger.Warn("failed to leave raft configuration gracefully, timeout")
		}
	}

	s.logger.Info("closing raft transport")
	if err := s.raftTransport.Close(); err != nil {
		return fmt.Errorf("failed to close raft TCP transport: %w", err)
	}
	s.logger.Info("closing raft store")
	if err := s.raftStore.Close(); err != nil {
		return fmt.Errorf("failed to close raft store: %w", err)
	}
	return nil
}

func (s *Server) isLeader() bool {
	return s.raft.State() == raft.Leader
}

func (s *Server) numVoters() (int, error) {
	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return 0, fmt.Errorf("failed to get the number of voters: %w", err)
	}
	config := configFuture.Configuration()
	var n int
	for _, server := range config.Servers {
		if server.Suffrage == raft.Voter {
			n++
		}
	}
	return n, nil
}

var (
	ErrNotFound = errors.New("not found")
	ErrEmptyKey = errors.New("an empty key")
)

func (s *Server) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.kvstore[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *Server) Set(key, value string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("non-leader can't set key=%s, value=%s", key, value)
	}
	b, err := json.Marshal(&command{
		Op:    SetOp,
		Key:   key,
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("[leader] failed to create a set command: %w", err)
	}
	if err := s.raft.Apply(b, s.options.timeout).Error(); err != nil {
		return fmt.Errorf("[leader] failed to apply a set command: %w", err)
	}
	return nil
}

func (s *Server) Delete(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("non-leader can't delete a value associated to %s", key)
	}
	b, err := json.Marshal(&command{
		Op:  DeleteOp,
		Key: key,
	})
	if err != nil {
		return fmt.Errorf("[leader] failed to create a delete command: %w", err)
	}
	if err := s.raft.Apply(b, s.options.timeout).Error(); err != nil {
		return fmt.Errorf("[leader] failed to apply a delete command: %w", err)
	}
	return nil
}

// Join register a server to a cluster, identified by serverId and located at advertise, to this server.
// The server must be ready to respond to Raft communications at that address.
func (s *Server) Join(serverId, advertise string) error {
	confFuture := s.raft.GetConfiguration()
	if err := confFuture.Error(); err != nil {
		return fmt.Errorf("failed to proceed a join request: %w", err)
	}

	for _, srv := range confFuture.Configuration().Servers {
		if isSameServer(srv, serverId, advertise) {
			s.logger.Info(fmt.Sprintf("server %s at %s has already been a member of a cluster", serverId, advertise))
			return nil
		}
		// If a server already exists with either the joining server's ID or address,
		// that server may need to be removed from the config first.
		if isServerExist(srv, serverId, advertise) {
			if err := s.raft.RemoveServer(srv.ID, 0, 0).Error(); err != nil {
				return fmt.Errorf("failed to remove an existing server %s at %s: %w", serverId, advertise, err)
			}
		}
	}

	if err := s.raft.AddVoter(raft.ServerID(serverId), raft.ServerAddress(advertise), 0, 0).Error(); err != nil {
		return fmt.Errorf("failed to add the given server to the cluster as a staging server: %w", err)
	}
	return nil
}

func isServerExist(s raft.Server, serverId, advertise string) bool {
	return s.ID == raft.ServerID(serverId) || s.Address == raft.ServerAddress(advertise)
}

func isSameServer(s raft.Server, serverId, advertise string) bool {
	return s.ID == raft.ServerID(serverId) && s.Address == raft.ServerAddress(advertise)
}
