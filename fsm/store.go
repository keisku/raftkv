package fsm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

// Store is a key-value store and behaves as a FSM.
type Store struct {
	dir     string
	addr    string
	kvstore kvstore
	mu      sync.Mutex
	raft    *raft.Raft
	logger  hclog.Logger
	options *Options
}

// NewStore initializes a store.
func NewStore(dir, addr string, l hclog.Logger, opt ...Option) *Store {
	opts := newOptions(opt...)
	return &Store{
		dir:     dir,
		addr:    addr,
		kvstore: make(kvstore),
		logger:  l,
		options: opts,
	}
}

// Open opens the store. If isSingle is set, and there are no existing peers,
// then this node becomes the first node, and therefore leader of the cluster.
func (s *Store) Open(ctx context.Context, serverId string, isSingle bool) error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to resolve a TCP address: %w", err)
	}

	tp, err := raft.NewTCPTransport(
		s.addr,
		tcpAddr,
		s.options.maxPool,
		s.options.timeout,
		os.Stderr,
	)
	if err != nil {
		return fmt.Errorf("failed to build a new TCP transport: %w", err)
	}

	ss, err := raft.NewFileSnapshotStore(s.dir, s.options.retain, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to build a new TCP transport: %w", err)
	}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(serverId)

	logStore, stableLogStore, err := newLogStore(filepath.Join(s.dir, "raft.db"))
	if err != nil {
		return fmt.Errorf("failed to create a log store: %s", err)
	}

	s.raft, err = raft.NewRaft(
		config,
		s,
		logStore,
		stableLogStore,
		ss,
		tp,
	)
	if err != nil {
		return fmt.Errorf("failed to construct a new Raft node: %w", err)
	}

	if isSingle {
		s.logger.Info("bootstraping the cluster")
		if err := s.raft.BootstrapCluster(raft.Configuration{
			Servers: []raft.Server{
				{ID: raft.ServerID(serverId), Address: tp.LocalAddr()},
			},
		}).Error(); err != nil {
			return fmt.Errorf("failed to bootstrap a cluster: %w", err)
		}
	}

	go func() {
		<-ctx.Done()
		s.logger.Info("closing a tcp transport")
		_ = tp.Close()
	}()

	return nil
}

// This function is for unit tests.
var newLogStore = func(path string) (raft.LogStore, raft.StableStore, error) {
	boltDB, err := raftboltdb.NewBoltStore(path)
	if err != nil {
		return nil, nil, err
	}
	return boltDB, boltDB, nil
}

var (
	ErrNotFound = errors.New("not found")
	ErrEmptyKey = errors.New("an empty key")
)

func (s *Store) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.kvstore[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *Store) Set(key, value string) error {
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

func (s *Store) Delete(key string) error {
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

// Join joins a node, identified by nodeID and located at addr, to this store.
// The node must be ready to respond to Raft communications at that address.
func (s *Store) Join(nodeId, addr string) error {
	confFuture := s.raft.GetConfiguration()
	if err := confFuture.Error(); err != nil {
		return fmt.Errorf("failed to proceed a join request: %w", err)
	}

	for _, srv := range confFuture.Configuration().Servers {
		if isSameServer(srv, nodeId, addr) {
			s.logger.Info(fmt.Sprintf("node %s at %s has already been a member of a cluster", nodeId, addr))
			return nil
		}
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if isServerExist(srv, nodeId, addr) {
			if err := s.raft.RemoveServer(srv.ID, 0, 0).Error(); err != nil {
				return fmt.Errorf("failed to remove an existing node %s at %s: %w", nodeId, addr, err)
			}
		}
	}

	if err := s.raft.AddVoter(raft.ServerID(nodeId), raft.ServerAddress(addr), 0, 0).Error(); err != nil {
		return fmt.Errorf("failed to add the given server to the cluster as a staging server: %w", err)
	}
	return nil
}

func isServerExist(s raft.Server, nodeId, addr string) bool {
	return s.ID == raft.ServerID(nodeId) || s.Address == raft.ServerAddress(addr)
}

func isSameServer(s raft.Server, nodeId, addr string) bool {
	return s.ID == raft.ServerID(nodeId) && s.Address == raft.ServerAddress(addr)
}
