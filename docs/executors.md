Executors
=========

dib supports multiple build executors. An executor is a platform able to run image builds and tests.
Unlike the build backends which can be explicitely chosen, the executor is automatically selected depending on the type 
of operation (build, test), and the executors configured in the configuration file.

**Build backend compatibility matrix**

| Executor   | BuildKit (recommended) | Docker |
|------------|------------------------|--------|
| Local      | ✔                      | ✔      |
| Docker     | ✗                      | ✗      |
| Kubernetes | ✔                      | ✗      |

## Local

Runs commands using the local exec system call. Use the `--local-only` flag to force the local executor.

## Docker

Runs commands in a docker container, using the `docker run` command.

## Kubernetes

Creates pods in a kubernetes cluster, using the kubernetes API. 
dib uses the current kube context, please make do

See an example configuration in the 
[configuration reference](configuration-reference.md) section.

When a build or test pod is stopped by the cluster rather than by its own workload (eviction, node drain,
deletion by someone else), dib exits with code `75` (`EX_TEMPFAIL`), provided every failed build or test was
disrupted this way. Any other failure exits with code `1`. CI can use it to retry the job, for instance with GitLab CI:

```yaml
retry:
  max: 2
  exit_codes: [75]
```

Deleting a pod by hand also counts as a disruption, so with such a retry rule the job runs again: to stop a build
for good, cancel the CI job instead of deleting its pod.
