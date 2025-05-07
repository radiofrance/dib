# DIB (Docker Image Builder) Example Configuration Guide

This guide explains how to configure and use the DIB example for building and pushing Docker images using Kubernetes.

## Overview

The example in this directory demonstrates how to use DIB to build Docker images using Buildkit in a Kubernetes cluster. The example includes:

- A Dockerfile for creating a custom Buildkit image
- A buildkitd-launcher script for starting the Buildkit daemon
- A .dib.yaml configuration file for configuring the build process

## Building and Pushing the Image

The first step is to build and push the custom Buildkit image defined in the Dockerfile:

```bash
# Navigate to the examples directory
cd examples

# Build the image
docker build -t <your-registry>/custome-buildkit:latest .

# Push the image to your registry
docker push <your-registry>/custome-buildkit:latest
```

Replace `<your-registry>` with your Docker registry URL. For example, using the registry from the .dib.yaml file:

```bash
docker build -t <your-registry>/custome-buildkit:latest .
docker push <your-registry>/custome-buildkit:latest
```

## Creating Docker Config Secret

DIB requires a Kubernetes secret containing Docker registry credentials to authenticate when pushing images. This secret is referenced by the `docker_config_secret` field in the .dib.yaml file.

To create this secret:

1. First, make sure you're logged in to your Docker registry:

```bash
docker login <your-registry>
```

2. Create a Kubernetes secret from your Docker config file:

```bash
kubectl create secret generic docker-config-prod \
  --from-file=config.json=$HOME/.docker/config.json \
  --namespace=default
```

This creates a secret named `docker-config-prod` in the `default` namespace, which matches the configuration in the .dib.yaml file.

## Creating Image Pull Secrets

Image pull secrets are used by Kubernetes to pull images from private registries. These secrets are referenced by the `image_pull_secrets` field in the .dib.yaml file.

To create an image pull secret:

```bash
kubectl create secret docker-registry image-pull-secret \
  --docker-server=<your-registry> \
  --docker-username=<your-username> \
  --docker-password=<your-password> \
  --docker-email=<your-email> \
  --namespace=default
```

Replace the placeholders with your registry information. For example:

```bash
kubectl create secret docker-registry image-pull-secret \
  --docker-server=europe-west9-docker.pkg.dev \
  --docker-username=_json_key \
  --docker-password="$(cat /path/to/your/service-account-key.json)" \
  --docker-email=your-email@example.com \
  --namespace=default
```

For Google Container Registry (GCR) or Artifact Registry, you typically use a service account key as the password.

## Configuring .dib.yaml

The .dib.yaml file in this directory contains the configuration for DIB. Key fields include:

- `registry_url`: The URL of your Docker registry
- `buildkit.context.s3`: Configuration for storing the build context in an S3 bucket
- `buildkit.executor.kubernetes`: Configuration for the Kubernetes executor
  - `namespace`: The Kubernetes namespace to use
  - `image`: The custom Buildkit image to use
  - `docker_config_secret`: The name of the Docker config secret
  - `image_pull_secrets`: The names of the image pull secrets

Modify these fields as needed for your environment.

## Additional Configuration

### AWS S3 Bucket for Build Context

If you're using an AWS S3 bucket for the build context (as configured in the .dib.yaml file), make sure:

1. The S3 bucket exists and is accessible
2. Your Kubernetes cluster has the necessary permissions to access the bucket
3. You've configured the AWS region correctly

### Kubernetes Namespace

Ensure the namespace specified in the .dib.yaml file exists in your Kubernetes cluster:

```bash
kubectl create namespace <namespace>
```

### Resource Limits

You can uncomment and modify the `container_override` section in the .dib.yaml file to set resource limits for the Buildkit container.

## Running the Example

Once everything is configured, you can use DIB to build and push Docker images:

```bash
dib build --backend buildkit --push
```

## Troubleshooting

- If you encounter authentication issues, check that your Docker config secret is correctly created and referenced in the .dib.yaml file
- If pods fail to start, check that the image pull secrets are correctly created and referenced
- Check the pod logs for more detailed error messages: `kubectl logs <pod-name> -n <namespace>`