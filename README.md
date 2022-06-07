DIB: Docker Image Builder
=========================

![CI Status](https://img.shields.io/github/workflow/status/radiofrance/dib/CI?label=CI&logo=github%20actions&logoColor=fff)
[![codecov](https://codecov.io/gh/radiofrance/dib/branch/main/graph/badge.svg)](https://codecov.io/gh/radiofrance/dib)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/radiofrance/dib?sort=semver)

DIB is a tool designed to help build multiple Docker images defined within a directory, possibly having dependencies
with one another.

## Features

- Build all your Docker images with a single command.
- Only build images when something has changed since last build.
- Supports dependencies between images, builds will be queued until all parent images are built.
- Run test suites on images, and ensure the tests pass before promoting images.
- Multiple build backends supported (Docker/BuildKit, Kaniko)
- Multiple executors supported (Shell, Docker, Kubernetes)

## How it works

DIB recursively parses all Dockerfiles found within a directory, and builds a dependency graph. It computes a unique
hash from each image build context and Dockerfile (plus the hashes from images dependencies if any). This hash is then
converted to human-readable tag, which will make the final image tag.

When an image build context, Dockerfile, or any parent image changes, the hash changes (as well as the human-readable
tag) and DIB knows the image needs to be rebuilt. If the tag is already present on the registry, DIB considers there is
nothing to do as the image has already been built and pushed. This mechanism allows to only build what is necessary.

Example with a simple directory structure:

```
debian
├── Dockerfile      # Image: debian-bullseye
└── nginx
    └── Dockerfile  # Image: nginx
```

The parent `debian-bullseye` image depends on an external image, not managed by DIB :

```dockerfile
# debian/Dockerfile
FROM debian:bullseye
LABEL name="debian-bullseye"
```

To figure out the name of the image to build, DIB uses the `name` label (`debian-bullseye` here). The image name is then
appended to the configured registry URL (we'll use `gcr.io/project` in examples). The target image DIB will build here
is `gcr.io/project/debian-bullseye`.

For the `nginx` image, we need to extend the `gcr.io/project/debian-bullseye` image :

```dockerfile
# debian/nginx/Dockerfile
FROM gcr.io/project/debian-bullseye:latest
LABEL name="nginx"
```

The `latest` tag is a placeholder to tell DIB it should use the latest `debian-bullseye` image built by DIB itself. DIB
will always use the latest built image, based on the current filesystem state. If the `debian-bullseye`
image changed, it will be rebuilt first, then `nginx` will also be rebuilt because it depends on it.

Use the `--placeholder-tag` option to change the name of the placeholder tag if you want to have a distinct tag name to
avoid confusion with the `latest` tag.

## Installation

### With Go install:

```
go install github.com/radiofrance/dib@latest
```

### Download binaries:

Binaries are available to download from the [GitHub releases](https://github.com/radiofrance/dib/releases) page.

## Usage

Check `dib --help` for command usage.

## Configuration

DIB can be configured either by command-line flags, environment variables or configuration file.

The command-line flags have the highest priority, then environment variables, then config file. This means you can set
default values in the configuration file, and then override with environment variables of command-line flags.

### Command-line flags

Example:

```shell
dib build --registry-url=gcr.io/project path/to/images/dir
```

### Environment variables

Environment variables must be prefixed by `DIB_` followed by the capitalized, snake_cased flag name.

Example:

```shell
export DIB_REGISTRY_URL=gcr.io/project
dib build path/to/images/dir
```

### Configuration file

A `.dib.yaml` config file is expected in the current working directory. You can change the file location with
the `--config` (`-c`) flag.

The YAML keys must be camelCased flag names.

Example

```yaml
# .dib.yaml
registryUrl: gcr.io/project
```

## License

dib is released under the [CeCILL V2.1 License](https://cecill.info/licences/Licence_CeCILL_V2.1-en.txt)
