package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/SirCodeKnight/kvstore/internal/api"
	"github.com/SirCodeKnight/kvstore/internal/metrics"
	"github.com/SirCodeKnight/kvstore/internal/raft"
	"github.com/SirCodeKnight/kvstore/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	cfgFile     string
	nodeID      string
	httpAddr    string
	raftAddr    string
	joinAddr    string
	dataDir     string
	bootstrap   bool
	storageType string
)

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "kvstore-server",
		Short: "KVStore - High-Performance Distributed Key-Value Store",
		Long: `KVStore is a professional, distributed key-value store inspired by Redis and DynamoDB,
designed for high performance and reliability in production environments.
Built with Go, it offers a robust solution for distributed storage with strong consistency guarantees.`,
		Run: runServer,
	}

	// Add flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.Flags().StringVar(&nodeID, "id", "", "unique node ID")
	rootCmd.Flags().StringVar(&httpAddr, "http-addr", "localhost:8080", "HTTP API address")
	rootCmd.Flags().StringVar(&raftAddr, "raft-addr", "localhost:7000", "Raft internal address")
	rootCmd.Flags().StringVar(&joinAddr, "join", "", "leader address to join")
	rootCmd.Flags().StringVar(&dataDir, "data-dir", "./data", "data directory")
	rootCmd.Flags().BoolVar(&bootstrap, "bootstrap", false, "bootstrap a new cluster")
	rootCmd.Flags().StringVar(&storageType, "storage", "memory", "storage type (memory or disk)")

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Read the config
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// Check for environment variables or config values for each flag
	if viper.GetString("id") != "" {
		nodeID = viper.GetString("id")
	}
	if viper.GetString("http-addr") != "" {
		httpAddr = viper.GetString("http-addr")
	}
	if viper.GetString("raft-addr") != "" {
		raftAddr = viper.GetString("raft-addr")
	}
	if viper.GetString("join") != "" {
		joinAddr = viper.GetString("join")
	}
	if viper.GetString("data-dir") != "" {
		dataDir = viper.GetString("data-dir")
	}
	if viper.GetBool("bootstrap") {
		bootstrap = viper.GetBool("bootstrap")
	}
	if viper.GetString("storage") != "" {
		storageType = viper.GetString("storage")
	}
}

func runServer(cmd *cobra.Command, args []string) {
	// Initialize config
	initConfig()

	// Create logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Validate required parameters
	if nodeID == "" {
		logger.Fatal("node ID is required")
	}

	// Create data directories
	raftDir := filepath.Join(dataDir, "raft")
	kvDir := filepath.Join(dataDir, "kv")
	os.MkdirAll(raftDir, 0755)
	os.MkdirAll(kvDir, 0755)

	// Create metrics
	metricsCollector := metrics.NewMetrics("kvstore")

	// Create storage
	var store storage.Storage
	if storageType == "disk" {
		store, err = storage.NewDiskStorage(kvDir)
		if err != nil {
			logger.Fatal("failed to create disk storage", zap.Error(err))
		}
	} else {
		store = storage.NewMemoryStorage()
	}

	// Create Raft node
	node, err := raft.NewNode(nodeID, raftDir, raftAddr, store, logger)
	if err != nil {
		logger.Fatal("failed to create Raft node", zap.Error(err))
	}

	// Bootstrap or join the cluster
	if bootstrap {
		logger.Info("bootstrapping cluster", zap.String("node_id", nodeID))
		if err := node.Bootstrap([]string{nodeID}); err != nil {
			logger.Fatal("failed to bootstrap cluster", zap.Error(err))
		}
	} else if joinAddr != "" {
		logger.Info("joining cluster", zap.String("leader_addr", joinAddr))
		if err := node.JoinCluster(joinAddr); err != nil {
			logger.Fatal("failed to join cluster", zap.Error(err))
		}
	}

	// Create API server
	server := api.NewServer(node, httpAddr, metricsCollector, logger)

	// Start API server in a goroutine
	go func() {
		if err := server.Run(); err != nil {
			logger.Fatal("failed to start API server", zap.Error(err))
		}
	}()

	logger.Info("server started",
		zap.String("node_id", nodeID),
		zap.String("http_addr", httpAddr),
		zap.String("raft_addr", raftAddr),
		zap.String("storage", storageType))

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Gracefully shutdown
	logger.Info("shutting down server")
	if err := node.Close(); err != nil {
		logger.Error("failed to close node", zap.Error(err))
	}
}