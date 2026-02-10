FROM alpine:latest@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

# This Dockerfile creates an image that runs buildkitd with containerd worker
# To run this container:
# docker run --privileged --pid=host \
#   -e CONTAINERD_ADDRESS=/run/containerd/containerd.sock \
#   -e BUILDKIT_VERSION=v0.12.0 \
#   <image_name>
#
# Note: buildkitd and buildctl will be installed on the host system after nsenter.
# You can specify a specific version of buildkit to install by setting BUILDKIT_VERSION.
# This container only supports rootful mode and will exit with an error if containerd is running in rootless mode.

# Ensure we're running as root
USER root

# Install required packages
RUN apk add --no-cache \
    wget \
    tar \
    util-linux

# Copy the buildkit-entrypoint script and make it executable
COPY buildkit-entrypoint.sh /usr/local/bin/buildkit-entrypoint.sh
RUN chmod +x /usr/local/bin/buildkit-entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/buildkit-entrypoint.sh"]
