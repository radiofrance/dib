#!/usr/bin/env bash

set -euo pipefail


root="$(cd "$(dirname "$0")" && pwd)"

readonly root

# Default timeout for tests
readonly timeout="30m"

# Change to the project root directory
cd "$root/.."


# Run tests with go test
go test ./cmd/... -timeout="$timeout" -p 1 -run "TestVersion" -v
