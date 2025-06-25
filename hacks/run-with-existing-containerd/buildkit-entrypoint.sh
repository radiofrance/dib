#!/bin/sh
set -eu

# Get containerd socket from environment variable
: ${CONTAINERD_ADDRESS:=/run/containerd/containerd.sock}
: ${BUILDKIT_VERSION:=latest}


echo "WARNING: This script should be executed only in rootful mode."

# Enter the host PID namespace and run the commands directly
exec nsenter -t 1 -m -u -n -i -- sh -c "
# Verify if containerd is in rootful mode
if grep -q 'rootless=true' /proc/1/cmdline || grep -q 'rootless=true' /proc/\$(pgrep containerd)/cmdline 2>/dev/null; then
  echo 'Error: This container only supports rootful mode, but containerd is running in rootless mode.'
  exit 1
fi

# Install buildkitd and buildctl if not already installed
if ! command -v buildkitd >/dev/null 2>&1 || ! command -v buildctl >/dev/null 2>&1; then
  echo 'Installing buildkit...'

  # Create temporary directory for installation
  TEMP_DIR=\$(mktemp -d)
  cd \$TEMP_DIR

  # Download and extract buildkit
  wget -q https://github.com/moby/buildkit/releases/download/${BUILDKIT_VERSION}/buildkit-${BUILDKIT_VERSION}.linux-amd64.tar.gz
  tar xzf buildkit-${BUILDKIT_VERSION}.linux-amd64.tar.gz

  # Install binaries
  cp bin/buildkitd bin/buildctl /usr/local/bin/
  chmod +x /usr/local/bin/buildkitd /usr/local/bin/buildctl

  # Clean up
  cd - >/dev/null
  rm -rf \$TEMP_DIR

  echo 'Buildkit installed successfully'
fi

# Run buildkitd with containerd worker
exec buildkitd \\
  --oci-worker=false \\
  --containerd-worker=true \\
  --containerd-worker-addr=${CONTAINERD_ADDRESS} \\
  --containerd-worker-namespace=default \\
  --containerd-worker-snapshotter=overlayfs \\
  $*
"
