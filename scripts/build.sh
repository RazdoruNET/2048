#!/bin/bash

set -e

echo "Building SOCKS5 DPI Proxy..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build for current platform
echo "Building for current platform..."
go build -o bin/proxy ./cmd/proxy

# Build for multiple platforms
echo "Building for multiple platforms..."
GOOS=linux GOARCH=amd64 go build -o bin/proxy-linux-amd64 ./cmd/proxy
GOOS=windows GOARCH=amd64 go build -o bin/proxy-windows-amd64.exe ./cmd/proxy
GOOS=darwin GOARCH=amd64 go build -o bin/proxy-darwin-amd64 ./cmd/proxy
GOOS=darwin GOARCH=arm64 go build -o bin/proxy-darwin-arm64 ./cmd/proxy

echo "Build completed!"
echo "Binaries created in bin/ directory:"
ls -la bin/

echo ""
echo "Usage:"
echo "  ./bin/proxy -listen :1080 -config configs/proxy.yaml"
