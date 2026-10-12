#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'
EXT_NAME='xru-syntax.vsix'

echo -e "${GREEN}[INFO]${NC} Cleaning up old extension..."
rm -rf $EXT_NAME

echo -e "${BLUE}[INFO]${NC} Building VS Code Extension..."

# Check for vsce
if ! command -v vsce &> /dev/null; then
    echo -e "${BLUE}[INFO]${NC} 'vsce' not found. Installing globally via xfpm..."
    xfpm install -g @vscode/vsce
fi

# Package the extension
cd "$(dirname "$0")"
vsce package --out $EXT_NAME

echo -e "${GREEN}[SUCCESS]${NC} Extension packaged: $EXT_NAME"

# Install into current VS Code instance if --current or -c is provided
if [[ "$*" == *"--current"* || "$*" == *"-c"* ]]; then
    echo -e "${BLUE}[INFO]${NC} Installing extension into VS Code..."
    code --install-extension "$EXT_NAME" --force
    echo -e "${GREEN}[SUCCESS]${NC} Extension successfully installed to current VS Code instance!"
fi
