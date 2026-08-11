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
    if [ "$ARCH" = "arm64" ]; then
        BINARY_NAME="aluminium-linux-arm64"
    else
        BINARY_NAME="aluminium-linux-amd64"
    fi
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

# Set up shell alias and env file
ENV_FILE="$ALUM_DIR/env"
if [ ! -f "$ENV_FILE" ]; then
    echo "alias al=\"aluminium\"" > "$ENV_FILE"
elif ! grep -q "alias al=" "$ENV_FILE"; then
    echo "alias al=\"aluminium\"" >> "$ENV_FILE"
fi

# Detect shell config file
SHELL_NAME="$(basename "${SHELL:-bash}")"
RC_FILE=""
if [ "$SHELL_NAME" = "zsh" ]; then
    RC_FILE="$HOME/.zshrc"
elif [ "$SHELL_NAME" = "bash" ]; then
    if [ -f "$HOME/.bash_profile" ]; then
        RC_FILE="$HOME/.bash_profile"
    else
        RC_FILE="$HOME/.bashrc"
    fi
fi

ALIAS_LINE="alias al=\"aluminium\""
PATH_LINE="export PATH=\"\$PATH:\$HOME/.aluminium/bin\""

if [ -n "$RC_FILE" ]; then
    if ! grep -q "alias al=" "$RC_FILE" 2>/dev/null; then
        echo "" >> "$RC_FILE"
        echo "# Added by Aluminium CLI installer" >> "$RC_FILE"
        echo "$PATH_LINE" >> "$RC_FILE"
        echo "$ALIAS_LINE" >> "$RC_FILE"
        echo "✓ Created alias 'al' for 'aluminium' in $RC_FILE"
    else
        echo "✓ Alias 'al' is configured in $RC_FILE"
    fi
else
    echo "To use the 'al' alias, add the following to your shell config file:"
    echo "  $PATH_LINE"
    echo "  $ALIAS_LINE"
fi

# Verify if path is in PATH
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo ""
    echo "WARNING: $BIN_DIR is not in your current PATH."
    echo "Run \`source $RC_FILE\` or open a new terminal for PATH and 'al' alias to take effect."
else
    echo "Aluminium CLI is ready to run!"
fi
