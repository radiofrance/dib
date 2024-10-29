Executors
=========

dib supports multiple build executors. An executor is a platform able to run image builds and tests.
Unlike the build backends which can be explicitely chosen, the executor is automatically selected depending on the type 
of operation (build, test), and the executors configured in the configuration file.

**Build backend compatibility matrix**

| Executor   | Docker | Kaniko |
|------------|--------|--------|
| Local      | ✔      | ✗      |
| Docker     | ✗      | ✔      |
| Kubernetes | ✗      | ✔      |

## Local

Runs commands using the local exec system call. Use the `--local-only` flag to force the local executor.

## Docker

Runs commands in a docker container, using the `docker run` command.

## Kubernetes

Creates pods in a kubernetes cluster, using the kubernetes API. 
dib uses the current kube context, please make do

See an example configuration in the 
[configuration reference](configuration-reference.md) section.
