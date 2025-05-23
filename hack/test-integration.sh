#!/usr/bin/env bash

set -euo pipefail


root="$(cd "$(dirname "$0")" && pwd)"

readonly root

# Default timeout for tests
readonly timeout="30m"

# Run tests with gotestsum for better output formatting
# Use the full path to gotestsum
$(go env GOPATH)/bin/gotestsum --format=testname --packages="$root"/../cmd/... -- -timeout="$timeout" -p 1 -run "TestVersion"
