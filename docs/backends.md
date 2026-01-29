Build Backends
==============

The build backend is a software or service responsible for actually building the images. dib itself is not capable of
building images, it delegates this part to the build backend.

dib supports multiple build backends. Currently, available backends are `docker` and `buildkit`. You can select the 
backend to use with the `--backend` option. `buildkit` is now the recommended and default backend.

**Executor compatibility matrix**

| Backend  | Local | Docker | Kubernetes |
|----------|-------|--------|------------|
| Docker   | ✔     | ✗      | ✗          |
| BuildKit | ✔     | ✗      | ✔          |

## Docker

The `docker` backend uses [Docker](https://www.docker.com/) behind the scenes, and runs `docker build` You need to have 
the Docker CLI installed locally to use this backend.

**Authentication**

The Docker Daemon requires authentication to pull and push images from private registries. Run the 
[`docker login`](https://docs.docker.com/engine/reference/commandline/login/) command to authenticate.

Authentication settings are stored in a `config.json` file located by default in `$HOME/.docker/`.
If you need to provide a different configuration, you can set the `DOCKER_CONFIG` variable to the path to another 
directory, which should contain a `config.json` file.

**Remote Daemon**

If you want to set a custom docker daemon host, you can set the `DOCKER_HOST` environment variable. The builds will then
run on the remote host instead of using the local Docker daemon.

**BuildKit**

If available, dib will try to use the BuildKit engine to build images, which is faster than the default Docker
build engine.

## BuildKit

[BuildKit](https://github.com/moby/buildkit) is a toolkit for converting source code to build artifacts in an efficient, expressive and repeatable manner. It provides a more efficient, cache-aware, and concurrent build engine compared to the traditional Docker build.

**Authentication**

BuildKit uses the same authentication mechanism as Docker. Run the [`docker login`](https://docs.docker.com/engine/reference/commandline/login/) command to authenticate with your registry.

**Local Builds**

For local builds, BuildKit requires the `buildctl` binary to be installed on your system and `buildkitd` daemon to be running. You can install BuildKit by following the instructions in the [official documentation](https://github.com/moby/buildkit#quick-start).

**Kubernetes Builds**

For Kubernetes builds, dib will create a pod with the BuildKit image and execute the build inside it. This requires proper configuration of Kubernetes access and Docker registry credentials.

**BuildKit Host**

You can specify a custom BuildKit daemon host using the `--buildkit-host` option or by setting the `BUILDKIT_HOST` environment variable.

See the `buildkit` section in the [configuration reference](configuration-reference.md).
