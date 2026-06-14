#!/usr/bin/env bash
set -e

# Build the Go contract to WASM using TinyGo
# Ensure tinygo is installed in PATH

tinygo build -target wasi -gc=leaking -opt=0 -no-debug -o main.wasm ./contract

echo "✅ Build completed: main.wasm"
