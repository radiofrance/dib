# Check for new release: https://github.com/GoogleContainerTools/kaniko/tags
ARG KANIKO_VERSION=v1.6.0
# We are using official Docker image as base
FROM gcr.io/kaniko-project/executor:${KANIKO_VERSION} as kaniko_artifacts

FROM eu.gcr.io/my-test-repository/bullseye:v1
LABEL name="kaniko"
LABEL version="16"
