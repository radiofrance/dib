Quickstart Guide
================

This guide will show you the basics of dib. You will build a set of images locally using the local docker daemon.

## Prerequisites

Before using dib, ensure you have the following dependencies installed:

- [Docker](https://www.docker.com/) for building images on your local computer.
- [Graphviz](https://graphviz.org/) for generating visual representation of the dependency graph (optional)
- [Goss](https://github.com/goss-org/goss) for testing images after build (optional)
- [Trivy](https://aquasecurity.github.io/trivy) for scanning images for vulnerabilites (optional)

Then, you need to install the dib command-line by following the [installation guide](install.md).

Make sure you have authenticated access to an OCI registry, in this guide we'll assume it is `registry.example.com`.

## Directory structure

Let's create a root directory containing 2 Dockerfiles in their own subdirectories.
The structure will look like:
```
docker/
├── base
|   └── Dockerfile
└── child
    └── Dockerfile
```

Now create the dockerfile for the `base` image:
```dockerfile
# docker/base/Dockerfile
FROM alpine:latest

LABEL name="base"
```

The "name" label is mandatory, it is used by dib to name the current image, by appending the value of the label to the 
registry URL. In this case, the image name is `registry.example.com/base`.

Then, create the dockerfile for the `child` image, which extends the `base` image:
```dockerfile
# docker/child/Dockerfile
FROM registry.example.com/base:latest

LABEL name="child"
```

/// admonition | Tip
    type: tip

The directory structure does not matter to dib. It builds the graph of dependencies based on the FROM statements.
You can have either flat directory structure like shown above, or embed child images context directories
in the parent context.
///

## Configuration

See the [configuration section](configuration.md) 

For this guide, we'll use a configuration file as it is the more convenient way for day-to-day usage.

Let's create a `.dib.yaml` next to the docker build directory:
```
docker/
├── base/
├── child/
└── .dib.yaml
```

Edit the file to set the registry name, used to pull and push dib-managed images.
```yaml
registry_url: registry.example.com
```

You can check everything is correct by running `dib list`:
```console
$ dib list
Using config file: docs/examples/.dib.yaml
  NAME   HASH
  base   august-berlin-blossom-magnesium
  child  gee-minnesota-maryland-robin
```

You should get the output containing the list of images that dib has discovered.

## Building the images

When you have all your images definitions in the build directory and configuration set up, you can proceed to building 
the images:
```console
$ dib build
...
```

When it's done, you can run the build command again, and you'll see that dib does nothing as long as the Dockerfiles 
remain unchanged.

When you are ready to promote the images to `latest`, run:
```console
$ dib build --release
```
