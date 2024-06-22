# Base image used to test DIB html report
# Build OK, Test OK
FROM debian:bookworm@sha256:a92ed51e0996d8e9de041ca05ce623d2c491444df6a535a566dabd5cb8336946
LABEL name="base-debian-bookworm"

ENV DOCKER_USER_ID=1664
ENV DOCKER_USER=debian

USER root

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN set +x \
  # Create dedicated user
  && groupadd --gid ${DOCKER_USER_ID} ${DOCKER_USER} \
  && useradd \
    --comment "${DOCKER_USER} user" \
    --uid ${DOCKER_USER_ID} \
    --gid ${DOCKER_USER_ID} \
    --create-home \
    --home-dir /home/${DOCKER_USER} \
    --shell /bin/bash \
    ${DOCKER_USER}

USER ${DOCKER_USER}
