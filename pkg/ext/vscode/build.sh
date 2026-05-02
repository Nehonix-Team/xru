#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}[INFO]${NC} Building VS Code Extension..."

# Check for vsce
if ! command -v vsce &> /dev/null; then
    echo -e "${BLUE}[INFO]${NC} 'vsce' not found. Installing globally via xfpm..."
    xfpm install -g @vscode/vsce
fi

# Package the extension
vsce package --out xru-syntax.vsix

echo -e "${GREEN}[SUCCESS]${NC} Extension packaged: pkg/ext/vscode/xru-syntax.vsix"
