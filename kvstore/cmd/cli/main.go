package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	serverAddr string
	ttl        int
)

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "kvstore-cli",
		Short: "KVStore CLI - Command Line Interface for KVStore",
		Long: `KVStore CLI provides a simple command-line interface to interact with 
the KVStore distributed key-value store.`,
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "http://localhost:8080", "server address")

	// Get command
	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a value by key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			resp, err := http.Get(fmt.Sprintf("%s/v1/kv/%s", serverAddr, key))
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Error: %s (HTTP %d)\n", string(body), resp.StatusCode)
				os.Exit(1)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				os.Exit(1)
			}

			fmt.Println(string(body))
		},
	}

	// Set command
	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a key-value pair",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			value := args[1]

			url := fmt.Sprintf("%s/v1/kv/%s", serverAddr, key)
			if ttl > 0 {
				url = fmt.Sprintf("%s?ttl=%d", url, ttl)
			}

			req, err := http.NewRequest("PUT", url, bytes.NewBufferString(value))
			if err != nil {
				fmt.Printf("Error creating request: %v\n", err)
				os.Exit(1)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Error: %s (HTTP %d)\n", string(body), resp.StatusCode)
				os.Exit(1)
			}

			fmt.Println("OK")
		},
	}
	setCmd.Flags().IntVar(&ttl, "ttl", 0, "time-to-live in seconds (0 means no expiration)")

	// Delete command
	deleteCmd := &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a key",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]

			req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/kv/%s", serverAddr, key), nil)
			if err != nil {
				fmt.Printf("Error creating request: %v\n", err)
				os.Exit(1)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Error: %s (HTTP %d)\n", string(body), resp.StatusCode)
				os.Exit(1)
			}

			fmt.Println("OK")
		},
	}

	// Keys command
	keysCmd := &cobra.Command{
		Use:   "keys",
		Short: "List all keys",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := http.Get(fmt.Sprintf("%s/v1/kv", serverAddr))
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Error: %s (HTTP %d)\n", string(body), resp.StatusCode)
				os.Exit(1)
			}

			var result struct {
				Keys []string `json:"keys"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Printf("Error parsing response: %v\n", err)
				os.Exit(1)
			}

			if len(result.Keys) == 0 {
				fmt.Println("No keys found")
				return
			}

			for _, key := range result.Keys {
				fmt.Println(key)
			}
		},
	}

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show server status",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := http.Get(fmt.Sprintf("%s/v1/raft/status", serverAddr))
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Error: %s (HTTP %d)\n", string(body), resp.StatusCode)
				os.Exit(1)
			}

			var status struct {
				Leader   string `json:"leader"`
				IsLeader bool   `json:"is_leader"`
				NodeID   string `json:"node_id"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
				fmt.Printf("Error parsing response: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Node ID: %s\n", status.NodeID)
			fmt.Printf("Leader: %s\n", status.Leader)
			fmt.Printf("Is Leader: %v\n", status.IsLeader)
		},
	}

	// Add commands to root
	rootCmd.AddCommand(getCmd, setCmd, deleteCmd, keysCmd, statusCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}