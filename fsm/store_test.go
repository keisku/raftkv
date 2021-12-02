package fsm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
)

func setupStore(ctx context.Context, serverId, addr string, isSingle bool) (*Store, error) {
	inmemstore := raft.NewInmemStore()
	newStableStore = func(path string) (raft.StableStore, error) {
		return inmemstore, nil
	}
	newLogStore = func(store raft.LogStore) (raft.LogStore, error) {
		return inmemstore, nil
	}
	dir := filepath.Join("raftkv.d", serverId)
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0700)
	go func() {
		<-ctx.Done()
		_ = os.RemoveAll(dir)
	}()
	s := NewStore(dir, addr, hclog.New(hclog.DefaultOptions))
	if err := s.Open(ctx, serverId, isSingle); err != nil {
		return nil, err
	}
	if isSingle {
		return s, nil
	}

	if err := s.Join(serverId, addr); err != nil {
		return nil, err
	}
	return s, nil
}

func TestSingleStoreAllOps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a leader store and a cluster.
	s, err := setupStore(ctx, "leader", "localhost:50000", true)
	assert.Nil(t, err)
	time.Sleep(3 * time.Second) // wait for a server ready.

	// Set an empty key.
	assert.Error(t, ErrEmptyKey, s.Set("", "value"))

	// Set a key-value.
	assert.Nil(t, s.Set("key", "value"))

	v, err := s.Get("key")
	assert.Nil(t, err)
	assert.Equal(t, "value", v)

	// Delete a key-value.
	assert.Nil(t, s.Delete("key"))

	// Ensure a key-value is deleted.
	v, err = s.Get("key")
	assert.Equal(t, ErrNotFound, err)
}
