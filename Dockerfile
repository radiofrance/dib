FROM ubuntu:latest AS builder

RUN apt-get update && apt-get install -y \
    curl \
    git \
    make

# Set up Go
RUN curl -L https://go.dev/dl/go1.24.3.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /app
COPY . .

RUN make build

# Second stage: create a minimal image with just the binary
FROM ubuntu:latest

RUN apt-get update && apt-get install -y \
    curl \
    git \
    iptables \
    ca-certificates \
    gnupg

RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
RUN echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
RUN apt-get update && apt-get install -y docker-ce-cli

RUN mkdir -p /tmp/buildkit && \
    curl -sSL https://github.com/moby/buildkit/releases/download/v0.12.5/buildkit-v0.12.5.linux-amd64.tar.gz | \
    tar -xz -C /tmp/buildkit && \
    mv /tmp/buildkit/bin/* /usr/local/bin/ && \
    rm -rf /tmp/buildkit


RUN curl -L https://go.dev/dl/go1.24.3.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH=$PATH:/usr/local/go/bin

COPY --from=builder /app/dist/dib /usr/local/bin/dib

WORKDIR /app
COPY --from=builder /app/hack /app/hack
COPY --from=builder /app/cmd /app/cmd
COPY --from=builder /app/go.mod /app/go.mod
COPY --from=builder /app/go.sum /app/go.sum
COPY --from=builder /app/pkg /app/pkg
COPY --from=builder /app/internal /app/internal

RUN chmod +x /app/hack/test-integration.sh

ENTRYPOINT ["/app/hack/test-integration.sh"]