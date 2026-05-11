#!/bin/bash

set -e

DEFAULT_LISTEN=":1080"
DEFAULT_CONFIG="configs/proxy.yaml"

LISTEN=${LISTEN:-$DEFAULT_LISTEN}
CONFIG=${CONFIG:-$DEFAULT_CONFIG}

echo "Starting SOCKS5 DPI Proxy..."
echo "Listen: $LISTEN"
echo "Config: $CONFIG"

# Create logs directory
mkdir -p logs

# Check if binary exists
if [ ! -f "bin/proxy" ]; then
    echo "Binary not found. Building..."
    ./scripts/build.sh
fi

# Start the proxy
./bin/proxy -listen "$LISTEN" -config "$CONFIG"
