SHORT_NAME ?= builder

include versioning.mk
DRYCC_REGISTRY ?= ${DEV_REGISTRY}

# container development environment variables
REPO_PATH := github.com/drycc/${SHORT_NAME}
DEV_ENV_IMAGE := ${DEV_REGISTRY}/drycc/go-dev
DEV_ENV_WORK_DIR := /opt/drycc/go/src/${REPO_PATH}
DEV_ENV_PREFIX := podman run --rm -v ${CURDIR}:${DEV_ENV_WORK_DIR} -w ${DEV_ENV_WORK_DIR} --entrypoint testdata/bin/fake-k8s
DEV_ENV_CMD := ${DEV_ENV_PREFIX} ${DEV_ENV_IMAGE}
PLATFORM ?= linux/amd64,linux/arm64

# SemVer with build information is defined in the SemVer 2 spec, but podman
# doesn't allow +, so we use -.
BINARY_DEST_DIR := rootfs/usr/bin
# Common flags passed into Go's linker.
LDFLAGS := "-s -w -X main.version=${VERSION}"
# Container Root FS
BINDIR := ./rootfs

bootstrap:
	${DEV_ENV_CMD} go mod vendor

# This illustrates a two-stage Podman build. podman-compile runs inside of
# the container environment. Other alternatives are cross-compiling, doing
# the build as a `podman build`.
build-binary:
	CGO_ENABLED=0 go build -ldflags ${LDFLAGS} -o ${BINARY_DEST_DIR}/boot boot.go

build:
	${DEV_ENV_CMD} make build-binary

test: test-style test-unit

test-style:
	${DEV_ENV_CMD} lint

test-unit:
	${DEV_ENV_CMD} go test --race ./...

test-cover:
	${DEV_ENV_CMD} test-cover.sh

podman-build:
	podman build -t ${IMAGE} --build-arg LDFLAGS=${LDFLAGS} --build-arg CODENAME=${CODENAME} .
	podman tag ${IMAGE} ${MUTABLE_IMAGE}

check-kubectl:
	@if [ -z $$(which kubectl) ]; then \
		echo "kubectl binary could not be located"; \
		exit 2; \
	fi

deploy: check-kubectl podman-build podman-push
	kubectl --namespace=drycc patch deployment drycc-$(SHORT_NAME) --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value":"$(IMAGE)"}]'

.PHONY: bootstrap depup build podman-build test test-style test-unit test-cover deploy
