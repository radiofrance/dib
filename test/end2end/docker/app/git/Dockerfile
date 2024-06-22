# This image should not build
# Build KO, Test SKIP
FROM registry.localhost/base-debian-bookworm:dib_managed
LABEL name="app-git"

USER root

RUN \
   apt-get update \
   && apt-get install -y lorem
