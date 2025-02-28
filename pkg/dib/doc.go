// Package dib provides tools and utilities for building container images using different backends.
//
// The available backends are:
// - Docker: Uses Docker builder to build images (will be deprecated soon).
// - Kaniko: Builds images inside a container or Kubernetes cluster without requiring a Docker daemon (will be deprecated soon).
//
// The package includes functionalities for managing and executing builds, handling authentication, and configuring build environments.
//
//nolint:lll
package dib
