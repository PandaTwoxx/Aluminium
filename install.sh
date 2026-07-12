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

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

BINARY_NAME=""
if [ "$OS" = "darwin" ]; then
    if [ "$ARCH" = "arm64" ]; then
        BINARY_NAME="aluminium-darwin-arm64"
    else
        BINARY_NAME="aluminium-darwin-amd64"
    fi
elif [ "$OS" = "linux" ]; then
    BINARY_NAME="aluminium-linux-amd64"
else
    echo "Error: Unsupported operating system: $OS"
    exit 1
fi

DOWNLOAD_URL="https://github.com/PandaTwoxx/Aluminium/releases/latest/download/${BINARY_NAME}"

echo "Downloading $BINARY_NAME from GitHub..."
curl -L -o "$BIN_DIR/aluminium" "$DOWNLOAD_URL"
chmod +x "$BIN_DIR/aluminium"

echo "Success! Aluminium CLI installed to: $BIN_DIR/aluminium"
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
