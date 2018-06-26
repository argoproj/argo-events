
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist
PLUGIN_DIR=${DIST_DIR}/plugins

VERSION=$(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)
PLUGINS=$(shell find . \( -type d -and -path '*/signals/stream/builtin/*' \))

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.gitTreeState=${GIT_TREE_STATE}

# docker image publishing options
DOCKER_PUSH=false
IMAGE_NAMESPACE=argoproj
IMAGE_TAG=latest

ifeq (${DOCKER_PUSH},true)
ifndef IMAGE_NAMESPACE
$(error IMAGE_NAMESPACE must be set to push images (e.g. IMAGE_NAMESPACE=argoproj))
endif
endif

ifneq (${GIT_TAG},)
IMAGE_TAG=${GIT_TAG}
override LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif
ifdef IMAGE_NAMESPACE
IMAGE_PREFIX=${IMAGE_NAMESPACE}/
endif

# Build the project
.PHONY: all controller controller-image clean test

all: controller-image

# Sensor controller
controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor-controller ./cmd/sensor-controller

controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make controller

controller-image: controller-linux stream-plugins-linux
	docker build -t $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) -f ./controller/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) ; fi

# Plugins
stream-plugins:
	go build -v -ldflags '${LDFLAGS}' -o ${PLUGIN_DIR}/nats ./signals/stream/builtin/nats
	go build -v -ldflags '${LDFLAGS}' -o ${PLUGIN_DIR}/mqtt ./signals/stream/builtin/mqtt
	go build -v -ldflags '${LDFLAGS}' -o ${PLUGIN_DIR}/kafka ./signals/stream/builtin/kafka
	go build -v -ldflags '${LDFLAGS}' -o ${PLUGIN_DIR}/amqp ./signals/stream/builtin/amqp

stream-plugins-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make stream-plugins

test:
	go test $(shell go list ./... | grep -v /vendor/) -race -short -v

coverage:
	go test -covermode=count -coverprofile=coverage.out $(shell go list ./... | grep -v /vendor/)
	go tool cover -func=coverage.out

clean:
	-rm -rf ${CURRENT_DIR}/dist
