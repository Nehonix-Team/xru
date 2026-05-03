#!/bin/bash

# Extract version from argument or from lib_version.go
if [ -z "$1" ]; then
    version=$(grep "BinVersion =" internal/utils/lib_version.go | cut -d '"' -f 2)
else
    version="$1"
fi

# Ensure version starts with 'v'
if [[ ! $version == v* ]]; then
    version="v$version"
fi

if [ -z "$version" ] || [ "$version" == "v" ]; then
    echo "Error: Version is required or could not be detected"
    exit 1
fi

echo "Indexing $version on Go proxy..."
GOPROXY=proxy.golang.org go list -m github.com/Nehonix-Team/xru@$version
