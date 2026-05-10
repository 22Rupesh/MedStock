#!/bin/bash
set -e

echo "Building Go application..."
go build -o api ./cmd/api

echo "Build complete. Starting server..."
./api