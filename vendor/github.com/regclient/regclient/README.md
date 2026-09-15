# regclient

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/regclient/regclient/Go?label=Go%20build)
![GitHub Workflow Status](https://img.shields.io/github/workflow/status/regclient/regclient/Docker?label=Docker%20build)
![GitHub](https://img.shields.io/github/license/regclient/regclient)
[![Go Reference](https://pkg.go.dev/badge/github.com/regclient/regclient.svg)](https://pkg.go.dev/github.com/regclient/regclient)

Client interface for the registry API.
This includes `regctl` for a command line interface to manage registries.

![regctl demo](docs/demo.gif)

## regclient Features

- Provides a client interface to interacting with registries.
- Images may be inspected without pulling the layers, allowing quick access to the image manifest and configuration.
- Tags may be listed for a repository.
- Repositories may be listed from a registry (if supported).
- Copying an image only pulls layers when needed, allowing images to be quickly retagged or promoted across repositories.
- Multi-platform images are supported, allowing all platforms to be copied between registries.
- Digest tags used by projects like sigstore/cosign are supported, allowing signature, attestation, and SBOM metadata to be copied with the image.
- OCI subject/referrers is supported for the standardized replacement of the "digest tags".
- Digests may be queried for a tag without pulling the manifest.
- Rate limits may be queried from the registry without pulling an image (useful for Docker Hub).
- Images may be imported and exported to both OCI and Docker formatted tar files.
- OCI Layout is supported for copying images to and from a local directory.
- Delete APIs have been provided for tags, manifests, and blobs (the tag deletion will only delete a single tag even if multiple tags point to the same digest).
- Registry logins are imported from docker when available
- Self signed, insecure, and http-only registries are all supported.
- Requests will retry and fall back to chunked uploads when network issues are encountered.

## regctl Features

`regctl` is a CLI interface to the `regclient` library.
In addition to the features listed for `regclient`, `regctl` adds the following abilities:

- Formatting output with templates.
- Push and pull arbitrary artifacts.

## regsync features

`regsync` is an image mirroring tool.
It will copy images between two locations with the following additional features:

- Uses a yaml configuration.
- The `regclient` copy is used to only pull needed layers, supporting multi-platform, and additional metadata.
- Can use user's docker configuration for registry credentials.
- Ability to run on a cron schedule, one time synchronization, or only check for stale images.
- Ability to backup previous target image before overwriting.
- Ability to postpone mirror step when rate limit is below a threshold.
- Ability to mirror multiple images concurrently.

## regbot features

`regbot` is a scripting tool on top of the `regclient` API with the following features:

- Runs user provided scripts based on a yaml configuration.
- Scripts are written in Lua and executed directly in Go.
- Can run on a cron schedule or a one time execution.
- Dry-run option can be used for testing.
- Built-in functions include:
  - Repository list
  - Tag list
  - Image manifest (either head or get, and optional resolving multi-platform reference)
  - Image config (this includes the creation time, labels, and other details shown in a `docker image inspect`)
  - Image rate limit and a wait function to delay the script when rate limit remaining is below a threshold
  - Image copy
  - Manifest delete
  - Tag delete

## Development Status

This project is in active development.
Various Go APIs may change, but efforts will be made to provide aliases and stubs for any removed API.

## Installing

See the [installation options](docs/install.md).

## Usage

See the [project documentation](docs/README.md).
