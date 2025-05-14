package raft

import (
	"encoding/json"
	"io"

	"github.com/SirCodeKnight/kvstore/internal/storage"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

// FSM implements the raft.FSM interface for the key-value store
type FSM struct {
	store  storage.Storage
	logger *zap.Logger
}

// Apply applies a Raft log entry to the key-value store
func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		f.logger.Error("failed to unmarshal command", zap.Error(err))
		return err
	}

	switch cmd.Op {
	case "set":
		err := f.store.Set(cmd.Key, cmd.Value)
		if err != nil {
			f.logger.Error("failed to set value", zap.String("key", cmd.Key), zap.Error(err))
			return err
		}
		f.logger.Debug("set value", zap.String("key", cmd.Key))
		return nil

	case "delete":
		err := f.store.Delete(cmd.Key)
		if err != nil {
			f.logger.Error("failed to delete key", zap.String("key", cmd.Key), zap.Error(err))
			return err
		}
		f.logger.Debug("deleted key", zap.String("key", cmd.Key))
		return nil

	case "deleteAll":
		err := f.store.Clear()
		if err != nil {
			f.logger.Error("failed to clear store", zap.Error(err))
			return err
		}
		f.logger.Debug("cleared store")
		return nil

	default:
		err := json.Unmarshal(log.Data, &cmd)
		f.logger.Error("unknown command", zap.String("op", cmd.Op), zap.Error(err))
		return err
	}
}

// Snapshot returns a snapshot of the key-value store
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.logger.Debug("creating snapshot")
	
	// Get all keys
	keys := f.store.Keys()
	
	// Create a map to hold all key-value pairs
	data := make(map[string]storage.Value, len(keys))
	
	// Populate the map
	for _, key := range keys {
		value, err := f.store.Get(key)
		if err == nil {
			data[key] = value
		}
	}
	
	return &fsmSnapshot{data: data}, nil
}

// Restore restores the key-value store from a snapshot
func (f *FSM) Restore(rc io.ReadCloser) error {
	f.logger.Debug("restoring from snapshot")
	
	// Clear the store first
	if err := f.store.Clear(); err != nil {
		f.logger.Error("failed to clear store", zap.Error(err))
		return err
	}
	
	// Read the snapshot data
	var data map[string]storage.Value
	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		f.logger.Error("failed to decode snapshot", zap.Error(err))
		return err
	}
	
	// Restore each key-value pair
	for key, value := range data {
		if err := f.store.Set(key, value); err != nil {
			f.logger.Error("failed to restore key", zap.String("key", key), zap.Error(err))
			// Continue restoring other keys
		}
	}
	
	return nil
}

// fsmSnapshot implements the raft.FSMSnapshot interface
type fsmSnapshot struct {
	data map[string]storage.Value
}

// Persist writes the snapshot to the given sink
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := json.NewEncoder(sink).Encode(s.data)
	if err != nil {
		sink.Cancel()
		return err
	}
	
	return sink.Close()
}

// Release is a no-op
func (s *fsmSnapshot) Release() {}