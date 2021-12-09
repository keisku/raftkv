package kv

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/kei6u/raftkv/config"
	"github.com/stretchr/testify/assert"
)

func setupServer(ctx context.Context, serverId, addr string, isSingle bool) (*Server, error) {
	dir := filepath.Join("raftkv.d", serverId)
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0700)
	l := hclog.New(hclog.DefaultOptions)
	s := NewServer(NewStore(l), l, &config.Values{
		ServerId:      serverId,
		AdvertiseAddr: addr,
		InMemory:      isSingle,
	})
	if err := s.Start(); err != nil {
		return nil, err
	}
	if isSingle {
		return s, s.BootstrapCluster()
	}

	if err := s.Join(serverId, addr); err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = os.RemoveAll(dir)
		_ = s.Shutdown()
	}()
	return s, nil
}

func TestSingleStoreAllOps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a leader store and bootstrap a cluster.
	s, err := setupServer(ctx, "leader", "localhost:50000", true)
	assert.Nil(t, err)
	time.Sleep(3 * time.Second) // wait for a server ready.

	// Set an empty key.
	assert.Error(t, ErrEmptyKey, s.ApplySetOp("", "value"))

	// Set a key-value.
	assert.Nil(t, s.ApplySetOp("key", "value"))

	v, err := s.ApplyGetOp("key")
	assert.Nil(t, err)
	assert.Equal(t, "value", v)

	// Delete a key-value.
	assert.Nil(t, s.ApplyDeleteOp("key"))

	// Ensure a key-value is deleted.
	v, err = s.ApplyGetOp("key")
	assert.Equal(t, ErrNotFound, err)
}
