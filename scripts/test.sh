#!/bin/bash

set -e

PROXY_HOST="127.0.0.1"
PROXY_PORT="1080"
TEST_DOMAIN="httpbin.org"
TEST_URL="http://httpbin.org/ip"

echo "Testing SOCKS5 DPI Proxy..."
echo "Proxy: $PROXY_HOST:$PROXY_PORT"
echo "Test domain: $TEST_DOMAIN"
echo ""

# Function to test proxy
test_proxy() {
    local test_url=$1
    local description=$2
    
    echo "Testing $description..."
    
    # Test with curl using SOCKS5 proxy
    if curl --socks5 "$PROXY_HOST:$PROXY_PORT" \
           --max-time 10 \
           --silent \
           --show-error \
           --write-out "%{http_code}\n" \
           "$test_url" > /tmp/test_result.txt 2>/tmp/test_error.txt; then
        
        http_code=$(cat /tmp/test_result.txt | tail -n1)
        if [ "$http_code" = "200" ]; then
            echo "✅ $description - SUCCESS (HTTP $http_code)"
            return 0
        else
            echo "❌ $description - FAILED (HTTP $http_code)"
            if [ -f /tmp/test_error.txt ]; then
                echo "   Error: $(cat /tmp/test_error.txt)"
            fi
            return 1
        fi
    else
        echo "❌ $description - FAILED"
        if [ -f /tmp/test_error.txt ]; then
            echo "   Error: $(cat /tmp/test_error.txt)"
        fi
        return 1
    fi
}

# Function to test direct connection (without proxy)
test_direct() {
    local test_url=$1
    local description=$2
    
    echo "Testing $description (direct)..."
    
    if curl --max-time 10 \
           --silent \
           --show-error \
           --write-out "%{http_code}\n" \
           "$test_url" > /tmp/test_direct.txt 2>/tmp/test_direct_error.txt; then
        
        http_code=$(cat /tmp/test_direct.txt | tail -n1)
        if [ "$http_code" = "200" ]; then
            echo "✅ $description - SUCCESS (HTTP $http_code)"
            return 0
        else
            echo "❌ $description - FAILED (HTTP $http_code)"
            return 1
        fi
    else
        echo "❌ $description - FAILED"
        return 1
    fi
}

# Check if proxy is running
echo "Checking if proxy is running..."
if ! nc -z "$PROXY_HOST" "$PROXY_PORT" 2>/dev/null; then
    echo "❌ Proxy is not running on $PROXY_HOST:$PROXY_PORT"
    echo "Please start the proxy first: ./scripts/run.sh"
    exit 1
fi

echo "✅ Proxy is running"
echo ""

# Test 1: Basic HTTP request through proxy
test_proxy "$TEST_URL" "Basic HTTP request"

# Test 2: HTTPS request through proxy
test_proxy "https://httpbin.org/ip" "HTTPS request"

# Test 3: Different domain
test_proxy "http://example.com" "Different domain (example.com)"

# Test 4: Direct connection for comparison
echo ""
echo "--- Direct Connection Tests ---"
test_direct "$TEST_URL" "Direct connection (no proxy)"

# Test 5: Check if proxy is bypassing correctly
echo ""
echo "--- DPI Bypass Tests ---"
test_proxy "https://www.google.com" "Google access (potential bypass)"

# Cleanup
rm -f /tmp/test_*.txt /tmp/test_*.error

echo ""
echo "Test completed!"
echo ""
echo "To use the proxy with other applications:"
echo "  curl --socks5 $PROXY_HOST:$PROXY_PORT http://example.com"
echo "  export ALL_PROXY=socks5://$PROXY_HOST:$PROXY_PORT"
echo ""
echo "Check proxy logs for detailed information about routing decisions."
