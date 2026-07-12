#!/bin/bash
set -e

# Target directories
ALUM_DIR="$HOME/.aluminium"
BIN_DIR="$ALUM_DIR/bin"

echo "=========================================="
echo "Installing/Updating Aluminium CLI..."
echo "=========================================="

# Create ~/.aluminium directory structure
mkdir -p "$BIN_DIR"
mkdir -p "$ALUM_DIR/install"
mkdir -p "$ALUM_DIR/build"

# Navigate to CLI directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$SCRIPT_DIR/AluminiumCLI"

if [ ! -d "$CLI_DIR" ]; then
    echo "Error: AluminiumCLI directory not found at $CLI_DIR"
    exit 1
fi

echo "Compiling Go CLI binary..."
cd "$CLI_DIR"
go build -o "$BIN_DIR/aluminium" ./cmd/aluminium

echo "Success! Aluminium CLI compiled and installed to: $BIN_DIR/aluminium"
echo "All application data and binaries now operate out of: $ALUM_DIR"

# Verify if path is in PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo ""
    echo "WARNING: $BIN_DIR is not in your current PATH."
    echo "To run the 'aluminium' command from any terminal, add the following to your shell config file (e.g. ~/.zshrc or ~/.bash_profile):"
    echo "  export PATH=\"\$PATH:\$HOME/.aluminium/bin\""
else
    echo "Aluminium CLI is in your PATH and ready to run!"
fi
