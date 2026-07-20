#!/bin/bash
set -euo pipefail

# Run bundle + generate and verify no files changed.
# Exits non-zero if generated files are out of date.

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

./scripts/bundle.sh
go generate ./...

if [ -n "$(git diff --name-only)" ]; then
    echo "ERROR: generated files are out of date. Please run 'make generate-go' and commit the changes."
    echo ""
    git diff --name-only
    exit 1
fi

echo "Generated files are up to date."
