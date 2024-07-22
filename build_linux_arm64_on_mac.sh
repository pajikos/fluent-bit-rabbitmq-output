#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Define the output file name
OUTPUT_FILE="out_rabbitmq.so"

# Define the source files
SOURCE_FILES="out_rabbitmq.go routing_key_validator.go routing_key_creator.go record_parser.go helper.go"

# Check if the cross-compiler is installed
if ! command -v aarch64-linux-musl-gcc &> /dev/null; then
    echo "Cross-compiler aarch64-linux-musl-gcc not found. Installing..."
    brew install FiloSottile/musl-cross/musl-cross
fi

# Set environment variables for cross-compilation
export GOOS=linux
export GOARCH=arm64
export CGO_ENABLED=1
export CC=aarch64-linux-musl-gcc

# Build the project
echo "Building the project for Linux ARM64..."
go build -buildmode=c-shared -o $OUTPUT_FILE $SOURCE_FILES

echo "Build completed successfully. Output file: $OUTPUT_FILE"

