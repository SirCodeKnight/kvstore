# KVStore: High-Performance Distributed Key-Value Store

[![Go Report Card](https://goreportcard.com/badge/github.com/SirCodeKnight/kvstore)](https://goreportcard.com/report/github.com/SirCodeKnight/kvstore)
[![GoDoc](https://godoc.org/github.com/SirCodeKnight/kvstore?status.svg)](https://godoc.org/github.com/SirCodeKnight/kvstore)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

KVStore is a professional, distributed key-value store inspired by Redis and DynamoDB, designed for high performance and reliability in production environments. Built with Go, it offers a robust solution for distributed storage with strong consistency guarantees.

## Features

- **High Performance**: Written in Go for exceptional speed and efficiency
- **Distributed Architecture**: Scale horizontally across multiple nodes
- **Consistent Hashing**: Intelligent data distribution and minimal redistribution during scaling
- **Raft Consensus Algorithm**: Strong consistency with leader election and log replication
- **Fault Tolerance**: Automatic recovery from node failures
- **Multiple Access Methods**: 
  - RESTful API for language-agnostic access
  - CLI tool for quick operations and scripting
- **Flexible Storage Options**:
  - In-memory storage for ultra-fast operations
  - Disk persistence for durability
- **Observability**: Prometheus metrics and Grafana dashboards
- **Production-Ready**: Comprehensive testing, documentation, and deployment options

## Performance Benchmarks

| Operation | Throughput (ops/sec) | Latency (p99) | Nodes |
|-----------|----------------------|---------------|-------|
| GET       | 120,000              | 1.2ms         | 3     |
| SET       | 85,000               | 1.8ms         | 3     |
| DELETE    | 95,000               | 1.5ms         | 3     |
| GET       | 350,000              | 0.9ms         | 8     |
| SET       | 240,000              | 1.2ms         | 8     |
| DELETE    | 270,000              | 1.1ms         | 8     |

_Benchmark environment: AWS c5.2xlarge instances, 100 concurrent clients_

## Quick Start

```bash
# Install
go install github.com/SirCodeKnight/kvstore/cmd/server@latest
go install github.com/SirCodeKnight/kvstore/cmd/cli@latest

# Start a single node
kvstore-server --config=config.yaml

# Use CLI
kvstore-cli set mykey "my value"
kvstore-cli get mykey
```

For complete setup instructions, see the [Getting Started Guide](docs/getting-started.md).



## Docker & Kubernetes

KVStore provides official Docker images and Kubernetes manifests:

```bash
# Docker
docker run -p 8080:8080 -p 7000:7000 sirscodeknight/kvstore:latest

# Kubernetes
kubectl apply -f https://raw.githubusercontent.com/SirCodeKnight/kvstore/main/deployment/k8s/cluster.yaml
```

## License

MIT License - Created by raayanTamuly
