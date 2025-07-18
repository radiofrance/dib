#!/usr/bin/env bash

set -euo pipefail


root="$(cd "$(dirname "$0")" && pwd)"

readonly root

readonly timeout="30m"

cd "$root/.."


# Start buildkitd if not already running
if ! buildctl debug workers &> /dev/null; then
    echo "Starting buildkitd..."
    mkdir -p /etc/buildkit
    buildkitd --oci-worker=true --containerd-worker=false &
    sleep 2
fi

go test ./cmd/... -timeout="$timeout" -p 1 -run "TestInteg" -v
