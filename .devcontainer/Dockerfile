FROM mcr.microsoft.com/devcontainers/base@sha256:1a00517e89d3075f8c9dc8d9911404de0ff8991a4d71c62068573e9b016a89da

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends \
  curl \
  yamllint \
  graphviz \
  && rm -rf /var/lib/apt/lists/*

# renovate: datasource=github-releases depName=moby/buildkit
ENV BUILDKIT_VERSION=v0.32.0
ENV BUILDKIT_URL=https://github.com/moby/buildkit/releases/download/${BUILDKIT_VERSION}/buildkit-${BUILDKIT_VERSION}.linux-amd64.tar.gz

# renovate: datasource=github-releases depName=goreleaser/goreleaser
ENV GORELEASER_VERSION=v2.14.3
ENV GORELEASER_URL=https://github.com/goreleaser/goreleaser/releases/download/${GORELEASER_VERSION}/goreleaser_Linux_x86_64.tar.gz

RUN echo "Downloading and installing BuildKit..." && \
  curl -L ${BUILDKIT_URL} -o buildkit.tar.gz && \
  mkdir buildkit_extracted && \
  tar -xzf buildkit.tar.gz -C buildkit_extracted && \
  echo "Copying BuildKit binaries..." && \
  cp buildkit_extracted/bin/buildctl /usr/local/bin/ && \
  cp buildkit_extracted/bin/buildkitd /usr/local/bin/ && \
  rm -rf buildkit_extracted buildkit.tar.gz && \
  \
  echo "Downloading and installing Goreleaser..." && \
  curl -L ${GORELEASER_URL} -o goreleaser.tar.gz && \
  mkdir goreleaser_extracted && \
  tar -xzf goreleaser.tar.gz -C goreleaser_extracted && \
  echo "Copying Goreleaser binary..." && \
  cp goreleaser_extracted/goreleaser /usr/local/bin/ && \
  rm -rf goreleaser_extracted goreleaser.tar.gz && \
  \
  echo "Verifying installations..." && \
  ls -l /usr/local/bin/buildctl /usr/local/bin/buildkitd /usr/local/bin/goreleaser && \
  echo "Verifying shell..." && \
  ls -l /bin/sh && /bin/sh -c "echo Shell test successful"
