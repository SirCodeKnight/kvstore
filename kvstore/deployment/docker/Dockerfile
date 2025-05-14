FROM golang:1.20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kvstore-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kvstore-cli ./cmd/cli

# Use a smaller image for the final build
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create a non-root user to run the application
RUN adduser -D -g '' kvstore

# Create necessary directories
RUN mkdir -p /data && chown -R kvstore:kvstore /data

# Copy the binaries from the builder stage
COPY --from=builder /app/kvstore-server /usr/local/bin/
COPY --from=builder /app/kvstore-cli /usr/local/bin/

# Set working directory
WORKDIR /data

# Use the non-root user
USER kvstore

# Expose the HTTP and Raft ports
EXPOSE 8080 7000

# Set volume for data persistence
VOLUME ["/data"]

# Command to run the server
ENTRYPOINT ["kvstore-server"]
CMD ["--id", "node1", "--http-addr", "0.0.0.0:8080", "--raft-addr", "0.0.0.0:7000", "--data-dir", "/data", "--bootstrap", "--storage", "disk"]