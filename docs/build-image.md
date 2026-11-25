Single Image Builds
===================

`dib build` builds a whole graph of images, and tags each one with a hash of its content. To build the image of an
application in its own CI pipeline, with tags you choose (a version, a commit SHA…), use `dib build-image` instead:

```shell
dib build-image --tag registry.example.org/app:1.0.0
```

It builds the Dockerfile of a single build context (the current directory by default, or the directory given as
argument), and tags the image with every `--tag`. Compared to `dib build`, there is no graph, no tag computed from
a hash, no tests and no HTML report: the image is always built.

```shell
dib build-image \
  --file docker/Dockerfile \
  --tag registry.example.org/app:1.0.0 \
  --tag registry.example.org/app:latest \
  --build-arg VERSION=1.0.0 \
  --target production \
  .
```

- `--file` is relative to the build context, and the Dockerfile must be inside it.
- The image gets the same [OCI labels](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
  as images built by `dib build` (revision, source, creation date…).
- Only the BuildKit [backend](backends.md) is supported:
    - by default, the image is built with the [Kubernetes executor](executors.md#kubernetes), configured in the
      `buildkit` section of the configuration file, and is always pushed;
    - with `--local-only`, the image is built by a local BuildKit daemon, and is pushed only with `--push`.

Example GitLab CI job, with the [container image](install.md) of dib:

```yaml
build-image:
  image:
    name: ghcr.io/radiofrance/dib:latest
    entrypoint: [""]
  script:
    - dib build-image --tag "$CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA" .
```

Secrets
-------

Credentials needed during the build, such as a token to download private dependencies, should not be copied into the
build context: they would end up in the uploaded context, and possibly in the image. Expose them as
[BuildKit secrets](https://docs.docker.com/build/building/secrets/) instead, with `--secret` and `--ssh`
(also available for `dib build`, and as the `secret` and `ssh` configuration keys):

```dockerfile
RUN --mount=type=secret,id=npmrc,target=/root/.npmrc npm ci
```

```shell
dib build-image --secret id=npmrc,env=NPMRC --tag registry.example.org/app:1.0.0
```

With `--local-only`, `env=` and `src=` refer to the environment variables and files of the machine running dib.

With the Kubernetes executor, they refer to the BuildKit pod, so the secrets never leave the cluster.
For instance, a Kubernetes Secret listed in `env_secrets` provides environment variables to the pod:

```yaml
buildkit:
  executor:
    kubernetes:
      env_secrets:
        - npm-credentials # A Kubernetes Secret with an NPMRC key
```

For `--ssh`, the key must be a file of the BuildKit pod, for instance mounted from a Kubernetes Secret with
`pod_template_override` and `container_override`.

See the [command reference](cmd/dib_build-image.md) for all the flags.
