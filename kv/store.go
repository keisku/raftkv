package kv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

// Store stores key-values and behaves FSM.
type Store struct {
	mu       sync.Mutex
	logger   hclog.Logger
	keyvalue map[string]string
}

// NewStore initializes the store.
func NewStore(logger hclog.Logger) *Store {
	return &Store{
		logger:   logger,
		keyvalue: map[string]string{},
	}
}

var _ raft.FSMSnapshot = (*Store)(nil)

func (s *Store) Persist(sink raft.SnapshotSink) error {
	b, err := json.Marshal(s.keyvalue)
	if err != nil {
		_ = sink.Cancel()
		return fmt.Errorf("cancel persisting a kvstore: %w", err)
	}
	if _, err := sink.Write(b); err != nil {
		_ = sink.Cancel()
		return fmt.Errorf("cancel persisting a kvstore: %w", err)
	}
	if err := sink.Close(); err != nil {
		return fmt.Errorf("failed to close a sink: %w", err)
	}
	return nil
}

func (s *Store) Release() {}

func (s *Store) clone() *Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := make(map[string]string, len(s.keyvalue))
	for k, v := range s.keyvalue {
		clone[k] = v
	}
	return &Store{keyvalue: clone, logger: s.logger}
}

var _ raft.FSM = (*Store)(nil)

type Command struct {
	Op    Op     `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Op string

const (
	SetOp    Op = "set"
	DeleteOp Op = "delete"
)

func (s *Store) Apply(rl *raft.Log) interface{} {
	var c Command
	if err := json.Unmarshal(rl.Data, &c); err != nil {
		s.logger.Warn("failed to parse data of raft log into command", "error", err)
		return nil
	}
	switch c.Op {
	case SetOp:
		s.applySetCommand(c.Key, c.Value)
		return nil
	case DeleteOp:
		s.applyDeleteCommand(c.Key)
		return nil
	default:
		s.logger.Warn(fmt.Sprintf("receives an unsupported operation: %s", c.Op))
		return nil
	}
}

func (s *Store) applySetCommand(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyvalue[key] = value
}

func (s *Store) applyDeleteCommand(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keyvalue, key)
}

func (s *Store) Snapshot() (raft.FSMSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clone(), nil
}

func (s *Store) Restore(rc io.ReadCloser) error {
	kv := make(map[string]string)
	if err := json.NewDecoder(rc).Decode(&kv); err != nil {
		return fmt.Errorf("failed to restore a FSM from a snapshot")
	}

	// Set the state from the snapshot.
	// Lock is not required.
	// see: https://pkg.go.dev/github.com/hashicorp/raft#FSM
	s.keyvalue = kv
	return nil
}

var ErrNotFound = errors.New("not found")

func (s *Store) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.keyvalue[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}
