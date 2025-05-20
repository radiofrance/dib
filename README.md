dib: Docker Image Builder
=========================

![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/radiofrance/dib?sort=semver)
![CI Status](https://img.shields.io/github/actions/workflow/status/radiofrance/dib/qa.yml?label=QA&logo=github-actions&logoColor=fff)
[![codecov](https://codecov.io/gh/radiofrance/dib/branch/main/graph/badge.svg)](https://codecov.io/gh/radiofrance/dib)
[![Go Report Card](https://goreportcard.com/badge/github.com/radiofrance/dib)](https://goreportcard.com/report/github.com/radiofrance/dib)

dib is a tool designed to help build multiple Docker images defined within a directory, possibly having dependencies
with one another, in a single command.

## Features

- **Incremental Builds:** Images are only built when something has changed since the last build, saving time and resources.
- **Dependency Resolution:** Supports dependencies between images. Builds will be queued until all parent images are built, ensuring a smooth and efficient build process.
- **Test Suites:** Run test suites on images and ensure the tests pass before promoting images to production.
- **Build Backends:** BuildKit is the recommended and default backend. Docker and Kaniko backends are deprecated and will be removed in a future release.
- **Execution Environments:** dib supports multiple executors, allowing you to build images using different environments such as Shell, Docker, or Kubernetes.

## Documentation

To get started with dib, please read the [documentation](https://radiofrance.github.io/dib).

## Contributing

We welcome contributions from the community! If you'd like to contribute to dib, please review our 
[contribution guidelines](https://github.com/radiofrance/dib/blob/main/CONTRIBUTING.md) for more information.

## License

dib is licensed under the [CeCILL V2.1 License](https://cecill.info/licences/Licence_CeCILL_V2.1-en.txt)

## Support

If you have any questions or encounter any issues, please feel free to 
[open an issue](https://github.com/radiofrance/dib/issues/new/choose) on GitHub.
