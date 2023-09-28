# Drycc Builder v2

[![Build Status](https://woodpecker.drycc.cc/api/badges/drycc/builder/status.svg)](https://woodpecker.drycc.cc/drycc/builder)
[![codecov](https://codecov.io/gh/drycc/builder/branch/main/graph/badge.svg)](https://codecov.io/gh/drycc/builder)
[![Go Report Card](https://goreportcard.com/badge/github.com/drycc/builder)](https://goreportcard.com/report/github.com/drycc/builder)
[![codebeat badge](https://codebeat.co/badges/0507e5d5-163b-4280-84ea-83bd2e0c8e41)](https://codebeat.co/projects/github-com-drycc-builder-main)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fdrycc%2Fbuilder.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fdrycc%2Fbuilder?ref=badge_shield)

Drycc - A Fork of Drycc Workflow

Drycc (pronounced DAY-iss) Workflow is an open source Platform as a Service (PaaS) that adds a developer-friendly layer to any [Kubernetes][k8s-home] cluster, making it easy to deploy and manage applications on your own servers.

For more information about Drycc Workflow, please visit the main project page at https://github.com/drycc/workflow.

We welcome your input! If you have feedback, please [submit an issue][issues]. If you'd like to participate in development, please read the "Development" section below and [submit a pull request][prs].

# About

The builder is primarily a git server that responds to `git push`es by executing either the `git-receive-pack` or `git-upload-pack` hook. After it executes one of those hooks, it takes the following high level steps in order:

1. Calls `git archive` to produce a tarball (i.e. a `.tar.gz` file) on the local file system
2. Starts a new [Kubernetes Pod](http://kubernetes.io/docs/user-guide/pods/) to build the code, according to the following rules:
  - If there is a `Dockerfile`, build the container with imagebuilder
  - Otherwise, use imagebuilder to build CNCF native buildpack
  - You can use `DRYCC_STACK` specifies the build type. Currently, it supports two types: `buildpack` and `container`

# Supported Off-Cluster Storage Backends

Builder currently supports the following off-cluster storage backends:

* GCS
* AWS/S3
* Azure
* Swift
* Alibaba OSS

# Development

The Drycc project welcomes contributions from all developers. The high level process for development matches many other open source projects. See below for an outline.

* Fork this repository
* Make your changes
* [Submit a pull request][prs] (PR) to this repository with your changes, and unit tests whenever possible
	* If your PR fixes any [issues][issues], make sure you write `Fixes #1234` in your PR description (where `#1234` is the number of the issue you're closing)
* The Drycc core contributors will review your code. After each of them sign off on your code, they'll label your PR with `LGTM1` and `LGTM2` (respectively). Once that happens, a contributor will merge it

## Container Based Development Environment

The preferred environment for development uses [the `go-dev` Container image](https://github.com/drycc/go-dev). The tools described in this section are used to build, test, package and release each version of Drycc.

To use it yourself, you must have [make](https://www.gnu.org/software/make/) installed and Container installed and running on your local development machine.

If you don't have Podman installed, please go to https://podman.io/ to install it.

After you have those dependencies, grab Go dependencies with `make bootstrap`, build your code with `make build` and execute unit tests with `make test`.

# Testing

The Drycc project requires that as much code as possible is unit tested, but the core contributors also recognize that some code must be tested at a higher level (functional or integration tests, for example).

The [end-to-end tests](https://github.com/drycc/workflow-e2e) repository has our integration tests. Additionally, the core contributors and members of the community also regularly [dogfood](https://en.wikipedia.org/wiki/Eating_your_own_dog_food) the platform. Since this particular component is at the center of much of the Drycc Workflow platform, we find it especially important to dogfood it.

## Running End-to-End Tests

Please see [README.md](https://github.com/drycc/workflow-e2e/blob/main/README.md) on the end-to-end tests repository for instructions on how to set up your testing environment and run the tests.

## Dogfooding

Please follow the instructions on the [official Drycc docs](http://docs-v2.readthedocs.org/en/latest/installing-workflow/installing-drycc-workflow/) to install and configure your Drycc Workflow cluster and all related tools, and deploy and configure an app on Drycc Workflow.


[s3-api-ref]: http://docs.aws.amazon.com/AmazonS3/latest/API/APIRest.html
[install-k8s]: http://kubernetes.io/gettingstarted/
[k8s-home]: http://kubernetes.io
[issues]: https://github.com/drycc/builder/issues
[prs]: https://github.com/drycc/builder/pulls
[v2.18]: https://github.com/drycc/workflow/releases/tag/v2.18.0


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fdrycc%2Fbuilder.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fdrycc%2Fbuilder?ref=badge_large)
