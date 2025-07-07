#!/bin/bash

# DevProxy Test Script
# This script demonstrates how to use DevProxy from WSL

echo "DevProxy Test Script"
echo "===================="
echo

# Check if token is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <api-token>"
    echo "Example: $0 abc123def456..."
    exit 1
fi

TOKEN="$1"
API_URL="http://127.0.0.1:2223/run"

echo "Testing connection to DevProxy..."
echo

# Test 1: Simple go version command
echo "Test 1: Running 'go version'"
curl -s -X POST "$API_URL" \
    -H "X-Admin-Token: $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "command": "go",
        "args": ["version"],
        "cwd": "C:\\Dev"
    }' | jq .

echo
echo "Test 2: PowerShell Get-Date"
curl -s -X POST "$API_URL" \
    -H "X-Admin-Token: $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "command": "powershell",
        "args": ["-Command", "Get-Date"],
        "cwd": "C:\\Dev"
    }' | jq .

echo
echo "Test 3: Testing rejected command (should fail)"
curl -s -X POST "$API_URL" \
    -H "X-Admin-Token: $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "command": "cmd",
        "args": ["/c", "echo test"],
        "cwd": "C:\\Dev"
    }'

echo
echo
echo "Test 4: Testing restricted path (should fail)"
curl -s -X POST "$API_URL" \
    -H "X-Admin-Token: $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "command": "go",
        "args": ["version"],
        "cwd": "C:\\Windows"
    }'

echo
echo
echo "Tests complete!"