package fsm

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/raft"
)

var _ raft.FSM = (*Server)(nil)

type command struct {
	Op    Op     `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Op string

const (
	SetOp    Op = "set"
	DeleteOp Op = "delete"
)

func (s *Server) Apply(rl *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(rl.Data, &c); err != nil {
		s.logger.Warn("failed to parse data of raft log into command", "error", err)
		return nil
	}
	switch c.Op {
	case SetOp:
		return s.applySetCommand(c.Key, c.Value)
	case DeleteOp:
		return s.applyDeleteCommand(c.Key)
	default:
		s.logger.Warn(fmt.Sprintf("receives an unsupported operation: %s", c.Op))
		return nil
	}
}

func (s *Server) applySetCommand(key, value string) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kvstore[key] = value
	return nil
}

func (s *Server) applyDeleteCommand(key string) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.kvstore, key)
	return nil
}

func (s *Server) Snapshot() (raft.FSMSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.kvstore.clone(), nil
}

func (s *Server) Restore(rc io.ReadCloser) error {
	kvs := make(kvstore)
	if err := json.NewDecoder(rc).Decode(&kvs); err != nil {
		return fmt.Errorf("failed to restore a FSM from a snapshot")
	}

	// Set the state from the snapshot.
	// Lock is not required.
	// see: https://pkg.go.dev/github.com/hashicorp/raft#FSM
	s.kvstore = kvs
	return nil
}

var _ raft.FSMSnapshot = (kvstore)(nil)

// kvstore implements raft.FSMSnapshot.
// see: https://pkg.go.dev/github.com/hashicorp/raft#FSMSnapshot
type kvstore map[string]string

func (s kvstore) clone() kvstore {
	clone := make(kvstore, len(s))
	for k, v := range s {
		clone[k] = v
	}
	return clone
}

func (s kvstore) Persist(sink raft.SnapshotSink) error {
	b, err := json.Marshal(s)
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

func (s kvstore) Release() {}
