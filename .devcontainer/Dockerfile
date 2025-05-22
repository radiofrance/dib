FROM mcr.microsoft.com/devcontainers/base

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends \
  curl \
  yamllint \
  graphviz \
  && rm -rf /var/lib/apt/lists/*

# renovate: datasource=github-releases depName=moby/buildkit
ENV BUILDKIT_VERSION=v0.22.0
ENV BUILDKIT_SHA256=e0d83a631a48f13232fcee71cbd913e6b11dbde0a45985fa1b99af27ab97086e
ENV BUILDKIT_URL=https://github.com/moby/buildkit/releases/download/${BUILDKIT_VERSION}/buildkit-${BUILDKIT_VERSION}.linux-amd64.tar.gz

# renovate: datasource=github-releases depName=goreleaser/goreleaser
ENV GORELEASER_VERSION=v2.9.0
ENV GORELEASER_SHA256=a066fcd713684abed0d750d7559f1a5d794fa2faa8e8f1ad2eecec8c373668a7
ENV GORELEASER_URL=https://github.com/goreleaser/goreleaser/releases/download/${GORELEASER_VERSION}/goreleaser_Linux_x86_64.tar.gz

RUN echo "Downloading and installing BuildKit..." && \
  curl -L ${BUILDKIT_URL} -o buildkit.tar.gz && \
  echo "${BUILDKIT_SHA256}  buildkit.tar.gz" | sha256sum -c - && \
  echo "BuildKit checksum verified." && \
  mkdir buildkit_extracted && \
  tar -xzf buildkit.tar.gz -C buildkit_extracted && \
  echo "Copying BuildKit binaries..." && \
  cp buildkit_extracted/bin/buildctl /usr/local/bin/ && \
  cp buildkit_extracted/bin/buildkitd /usr/local/bin/ && \
  rm -rf buildkit_extracted buildkit.tar.gz && \
  \
  echo "Downloading and installing Goreleaser..." && \
  curl -L ${GORELEASER_URL} -o goreleaser.tar.gz && \
  echo "${GORELEASER_SHA256}  goreleaser.tar.gz" | sha256sum -c - && \
  echo "Goreleaser checksum verified." && \
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
