FROM ubuntu:latest

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

# Entry point for tests
ENTRYPOINT ["./hack/test-integration.sh"]
