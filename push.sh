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
# Bypass cache by resolving directly from GitHub first
GOPROXY=direct go list -m github.com/Nehonix-Team/xru@$version && \
GOPROXY=proxy.golang.org go list -m github.com/Nehonix-Team/xru@$version

# Trigger pkg.go.dev indexation
MODULE="github.com/Nehonix-Team/xru@$version"
echo "Requesting pkg.go.dev indexation for $MODULE..."
status=$(curl -s -o /dev/null -w "%{http_code}" "https://pkg.go.dev/fetch/$MODULE")
if [ "$status" == "200" ] || [ "$status" == "303" ]; then
    echo "pkg.go.dev indexation triggered successfully (HTTP $status)"
    echo "Visit: https://pkg.go.dev/$MODULE"
else
    echo "pkg.go.dev responded with HTTP $status (may already be indexed or pending)"
fi
