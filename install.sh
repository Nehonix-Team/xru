#!/bin/bash
# ***************************************************************************
# * XRU (XyPriss Rule Unit) Installer Proxy
# * Fetches and executes the official Nehonix installation workflow
# * @license MIT
# ***************************************************************************
set -euo pipefail

OFFICIAL_INSTALLER="https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.sh"
ALT_INSTALLER="https://dll.nehonix.com/xru/install.sh"

if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$OFFICIAL_INSTALLER" 2>/dev/null | bash -s -- xru "$@" || \
    curl -fsSL "$ALT_INSTALLER" | bash -s -- "$@"
elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$OFFICIAL_INSTALLER" 2>/dev/null | bash -s -- xru "$@" || \
    wget -qO- "$ALT_INSTALLER" | bash -s -- "$@"
else
    echo "Error: curl or wget is required to run the installer." >&2
    exit 1
fi
