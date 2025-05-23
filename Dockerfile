FROM ubuntu:latest AS builder

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    make

# Set up Go
RUN curl -L https://go.dev/dl/go1.24.3.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH=$PATH:/usr/local/go/bin

# Copy project files
WORKDIR /app
COPY . .

# Build the project
RUN make build

# Second stage: create a minimal image with just the binary
FROM ubuntu:latest

# Install dependencies for testing (if needed)
RUN apt-get update && apt-get install -y \
    curl \
    git

# Install Go for testing
RUN curl -L https://go.dev/dl/go1.24.3.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH=$PATH:/usr/local/go/bin

# Install gotestsum for test formatting
RUN go install gotest.tools/gotestsum@latest && \
    mkdir -p /root/go/bin && \
    cp $(go env GOPATH)/bin/gotestsum /usr/local/bin/

# Copy the binary from the builder stage
COPY --from=builder /app/dist/dib /usr/local/bin/dib

# Copy necessary files for testing
WORKDIR /app
COPY --from=builder /app/hack /app/hack
COPY --from=builder /app/cmd /app/cmd

# Make the test script executable
RUN chmod +x /app/hack/test-integration.sh

# Entry point for tests
ENTRYPOINT ["/app/hack/test-integration.sh"]