#!/usr/bin/env bash

set -euo pipefail


root="$(cd "$(dirname "$0")" && pwd)"

readonly root

# Default timeout for tests
readonly timeout="30m"

# Change to the project root directory
cd "$root/.."

# Run tests with gotestsum for better output formatting
# Use the full path to gotestsum
$(go env GOPATH)/bin/gotestsum --format=testname --packages=./cmd/... -- -timeout="$timeout" -p 1 -run "TestVersion"
