SHORT_NAME ?= builder

include versioning.mk

# dockerized development environment variables
REPO_PATH := github.com/drycc/${SHORT_NAME}
DEV_ENV_IMAGE := drycc/go-dev
DEV_ENV_WORK_DIR := /go/src/${REPO_PATH}
DEV_ENV_PREFIX := docker run --rm -v ${CURDIR}:${DEV_ENV_WORK_DIR} -w ${DEV_ENV_WORK_DIR}
DEV_ENV_CMD := ${DEV_ENV_PREFIX} ${DEV_ENV_IMAGE}
PLATFORM ?= linux/amd64,linux/arm64

# SemVer with build information is defined in the SemVer 2 spec, but Docker
# doesn't allow +, so we use -.
BINARY_DEST_DIR := rootfs/usr/bin
# Common flags passed into Go's linker.
LDFLAGS := "-s -w -X main.version=${VERSION}"
# Docker Root FS
BINDIR := ./rootfs

DRYCC_REGISTRY ?= ${DEV_REGISTRY}

bootstrap:
	${DEV_ENV_CMD} go mod vendor

# This illustrates a two-stage Docker build. docker-compile runs inside of
# the Docker environment. Other alternatives are cross-compiling, doing
# the build as a `docker build`.
build-binary:
	CGO_ENABLED=0 go build -ldflags ${LDFLAGS} -o ${BINARY_DEST_DIR}/boot boot.go

build:
	${DEV_ENV_CMD} make build-binary

test: test-style test-unit

test-style:
	${DEV_ENV_CMD} lint --deadline

test-unit:
	${DEV_ENV_CMD} sh -c 'go test --race ./...'

test-cover:
	${DEV_ENV_CMD} test-cover.sh

docker-build:
	docker build ${DOCKER_BUILD_FLAGS} -t ${IMAGE} --build-arg LDFLAGS=${LDFLAGS} .
	docker tag ${IMAGE} ${MUTABLE_IMAGE}

docker-buildx:
	docker buildx build --platform ${PLATFORM} -t ${IMAGE} --build-arg LDFLAGS=${LDFLAGS} . --push

check-kubectl:
	@if [ -z $$(which kubectl) ]; then \
		echo "kubectl binary could not be located"; \
		exit 2; \
	fi

deploy: check-kubectl docker-build docker-push
	kubectl --namespace=drycc patch deployment drycc-$(SHORT_NAME) --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value":"$(IMAGE)"}]'

.PHONY: bootstrap depup build docker-build test test-style test-unit test-cover deploy
