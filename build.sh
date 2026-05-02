#!/bin/bash
# ***************************************************************************
# * XRU (XyPriss Rule Unit)
# * Cross-Platform Build Architecture
# * @license MIT
# ***************************************************************************

# Colors for premium terminal output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# Configuration
DIST_DIR="dist"
BINARY_NAME="xru"
ENTRY_POINT="./cmd/xru"
# Extract version from Go source
VERSION=$(grep "BinVersion =" internal/utils/lib_version.go | cut -d'"' -f2)

echo -e "\n${BLUE}${BOLD} XRU BUILD ENGINE ${NC} ${DIM}v${VERSION}${NC}"
echo -e "${DIM} ──────────────────────────────────────────────────${NC}\n"

# Check for compress flag
COMPRESS=false
if [[ "$*" == *"--compress"* ]]; then
    COMPRESS=true
    if ! command -v upx >/dev/null 2>&1; then
        echo -e " ${YELLOW}[WARN] UPX not found. Compression will be skipped.${NC}"
        COMPRESS=false
    fi
fi

# Multi-platform targets
# format: "os/arch"
TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

# Initialization
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Build Loop
for TARGET in "${TARGETS[@]}"; do
    IFS="/" read -r OS ARCH <<< "$TARGET"
    
    # Filename logic
    OUTPUT="${BINARY_NAME}-${OS}-${ARCH}"
    if [ "$OS" == "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi

    # Display progress
    printf "  ${CYAN}%-20s${NC} ${DIM}»${NC} " "Creating ${OS}/${ARCH}"
    
    # Build command
    # -s: Omit the symbol table and debug information
    # -w: Omit the DWARF symbol table
    if GOOS=$OS GOARCH=$ARCH go build -ldflags="-s -w" -o "${DIST_DIR}/${OUTPUT}" "$ENTRY_POINT"; then
        echo -e "${GREEN}[DONE]${NC}"
        
        # Compression phase
        if [ "$COMPRESS" = true ]; then
            printf "  ${MAGENTA}%-20s${NC} ${DIM}»${NC} " "UPX Compressing"
            if upx --best "${DIST_DIR}/${OUTPUT}" > /dev/null 2>&1; then
                echo -e "${MAGENTA}[OPTIMIZED]${NC}"
            else
                echo -e "${YELLOW}[SKIPPED]${NC}"
            fi
        fi
    else
        echo -e "${NC}[FAILED]${NC}"
        exit 1
    fi
done

echo -e "\n${GREEN}${BOLD} BUILD PIPELINE COMPLETE ${NC}"
echo -e "${DIM} Artifacts stored in: ./${DIST_DIR}${NC}\n"

ls -lh "$DIST_DIR" | grep "$BINARY_NAME" | awk '{print "  " $5 "\t" $9}'
echo ""
