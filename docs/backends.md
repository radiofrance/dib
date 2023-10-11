Build Backends
==============

The build backend is a software or service responsible for actually building the images. DIB itself is not capable of
building images, it delegates this part to the build backend.

DIB supports multiple build backends. Currently, available backends are `docker` and `kaniko`. You can select the 
backend to use with the `--backend` option.

**Executor compatibility matrix**

| Backend | Local | Docker | Kubernetes |
|---------|-------|--------|------------|
| Docker  | ✔     | ✗      | ✗          |
| Kaniko  | ✗     | ✔      | ✔          |

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

If available, DIB will try to use the BuildKit engine to build images, which is faster than the default Docker
build engine.

## Kaniko

[Kaniko](https://github.com/GoogleContainerTools/kaniko) offers a way to build container images inside a container 
or Kubernetes cluster, without the security tradeoff of running a docker daemon container with host privileges.

/// admonition | BuildKit
    type: info

As Kaniko must run in a container, it requires Docker when running local builds as it uses the `docker` executor.
///

See the `kaniko` section in the [configuration reference](configuration-reference.md).
