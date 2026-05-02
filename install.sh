#!/bin/bash
set -e

REPO="Nehonix-Team/xru"
BINARY_NAME="xru"

# Detect OS
OS_RAW="$(uname -s | tr '[:upper:]' '[:lower:]')"
OS=$OS_RAW
if [ "$OS_RAW" == "darwin" ]; then
    OS="darwin"
elif [ "$OS_RAW" == "linux" ]; then
    OS="linux"
else
    echo "❌ Unsupported OS: $OS_RAW"
    exit 1
fi

# Detect Architecture
ARCH_RAW="$(uname -m)"
ARCH=$ARCH_RAW
if [ "$ARCH_RAW" == "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH_RAW" == "aarch64" ] || [ "$ARCH_RAW" == "arm64" ]; then
    ARCH="arm64"
else
    echo "❌ Unsupported Architecture: $ARCH_RAW"
    exit 1
fi

# Get latest version from GitHub API
VERSION=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "❌ Error: Could not fetch latest version info."
    exit 1
fi

FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${FILENAME}"

echo "🚀 Installing ${BINARY_NAME} ${VERSION} for ${OS}/${ARCH}..."

# Download to a temporary location
TMP_BIN="/tmp/${BINARY_NAME}"
curl -sL -o "$TMP_BIN" "$URL"
chmod +x "$TMP_BIN"

# Move to path
DEST="/usr/local/bin/${BINARY_NAME}"

# Check for write permissions
if [ -w "/usr/local/bin" ]; then
    mv "$TMP_BIN" "$DEST"
else
    echo "🔒 /usr/local/bin is not writable. Requesting sudo..."
    sudo mv "$TMP_BIN" "$DEST"
fi

echo "✅ ${BINARY_NAME} successfully installed to ${DEST}"
xru version
