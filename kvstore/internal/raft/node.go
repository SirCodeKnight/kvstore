package raft

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/SirCodeKnight/kvstore/internal/storage"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"go.uber.org/zap"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
	leaderWaitDelay     = 100 * time.Millisecond
	maxLeaderWait       = 10 * time.Second
)

var (
	// ErrNotLeader is returned when a node attempts a leader-only operation
	ErrNotLeader = errors.New("not the leader")
	
	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("timeout")
)

// Command represents a command to be executed by the state machine
type Command struct {
	Op    string         `json:"op"`    // "set", "delete", "deleteAll"
	Key   string         `json:"key"`   // Key to operate on
	Value storage.Value  `json:"value"` // Value for set operation
}

// Node represents a node in the Raft cluster
type Node struct {
	ID          string
	RaftDir     string
	RaftBind    string
	logger      *zap.Logger
	store       storage.Storage // The actual key-value store
	raft        *raft.Raft      // The Raft consensus module
	fsm         *FSM            // The finite state machine
}

// NewNode creates a new Raft node
func NewNode(id, raftDir, raftBind string, store storage.Storage, logger *zap.Logger) (*Node, error) {
	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	}
	
	// Create node
	node := &Node{
		ID:       id,
		RaftDir:  raftDir,
		RaftBind: raftBind,
		logger:   logger,
		store:    store,
	}
	
	// Create the FSM for this node
	node.fsm = &FSM{
		store:  store,
		logger: logger,
	}
	
	// Create Raft directory if it doesn't exist
	if err := os.MkdirAll(raftDir, 0755); err != nil {
		return nil, err
	}
	
	// Create the Raft system
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)
	
	// Setup Raft communication
	addr, err := net.ResolveTCPAddr("tcp", raftBind)
	if err != nil {
		return nil, err
	}
	
	transport, err := raft.NewTCPTransport(raftBind, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}
	
	// Create the snapshot store
	snapshots, err := raft.NewFileSnapshotStore(raftDir, retainSnapshotCount, os.Stderr)
	if err != nil {
		return nil, err
	}
	
	// Create the log store and stable store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.bolt"))
	if err != nil {
		return nil, err
	}
	
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-stable.bolt"))
	if err != nil {
		return nil, err
	}
	
	// Instantiate the Raft system
	ra, err := raft.NewRaft(config, node.fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		return nil, err
	}
	node.raft = ra
	
	return node, nil
}

// Bootstrap configures the cluster, should only be called when bootstrapping a new cluster
func (n *Node) Bootstrap(nodes []string) error {
	// Convert nodes to Raft servers
	var servers []raft.Server
	for _, nodeID := range nodes {
		servers = append(servers, raft.Server{
			ID:      raft.ServerID(nodeID),
			Address: raft.ServerAddress(nodeID),
		})
	}
	
	// Bootstrap the cluster
	f := n.raft.BootstrapCluster(raft.Configuration{
		Servers: servers,
	})
	
	return f.Error()
}

// JoinCluster joins an existing Raft cluster
func (n *Node) JoinCluster(leaderAddr string) error {
	// Build a connection to the leader
	conn, err := net.DialTimeout("tcp", leaderAddr, raftTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	// Create join request
	cmd := Command{
		Op:  "join",
		Key: n.ID,
	}
	
	// Encode the command
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&cmd); err != nil {
		return err
	}
	
	// Send the join request
	if _, err := conn.Write(buf.Bytes()); err != nil {
		return err
	}
	
	// Wait for confirmation
	var response struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}
	
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return err
	}
	
	if !response.Success {
		return errors.New(response.Error)
	}
	
	return nil
}

// Get gets a key from the store
func (n *Node) Get(key string) (storage.Value, error) {
	return n.store.Get(key)
}

// Set sets a key in the store
func (n *Node) Set(key string, value storage.Value) error {
	if n.raft.State() != raft.Leader {
		return ErrNotLeader
	}
	
	cmd := Command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	
	b, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	
	f := n.raft.Apply(b, raftTimeout)
	return f.Error()
}

// Delete deletes a key from the store
func (n *Node) Delete(key string) error {
	if n.raft.State() != raft.Leader {
		return ErrNotLeader
	}
	
	cmd := Command{
		Op:  "delete",
		Key: key,
	}
	
	b, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	
	f := n.raft.Apply(b, raftTimeout)
	return f.Error()
}

// Keys returns all keys in the store
func (n *Node) Keys() []string {
	return n.store.Keys()
}

// WaitForLeader blocks until a leader is elected or timeout occurs
func (n *Node) WaitForLeader() error {
	timeout := time.Now().Add(maxLeaderWait)
	for time.Now().Before(timeout) {
		if leader := n.raft.Leader(); leader != "" {
			return nil
		}
		time.Sleep(leaderWaitDelay)
	}
	return ErrTimeout
}

// Leader returns the current leader's ID
func (n *Node) Leader() string {
	return string(n.raft.Leader())
}

// IsLeader returns true if this node is the leader
func (n *Node) IsLeader() bool {
	return n.raft.State() == raft.Leader
}

// Close closes the node
func (n *Node) Close() error {
	if n.raft != nil {
		future := n.raft.Shutdown()
		if err := future.Error(); err != nil {
			return err
		}
	}
	
	if n.store != nil {
		return n.store.Close()
	}
	
	return nil
}