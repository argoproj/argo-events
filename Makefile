PACKAGE=github.com/argoproj/argo-events
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist

VERSION=$(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

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

.PHONY: protogen
protogen:
	./hack/generate-proto.sh

.PHONY: clientgen
clientgen:
	./hack/update-codegen.sh

.PHONY: openapigen
openapi-gen:
	./hack/update-openapigen.sh

.PHONY: codegen
codegen: clientgen openapigen protogen

# this is the default stream service
STREAM=nats

# Build the project images
.DELETE_ON_ERROR:
all: controller-image artifact-image calendar-image resource-image webhook-image stream-image

.PHONY: all controller controller-image clean test

# Sensor controller
controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor-controller ./cmd/sensor-controller

controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make controller

controller-image: controller-linux
	docker build -t $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) -f ./controller/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) ; fi

# signal microservice binaries
artifact:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/artifact-signal ./signals/artifact/micro

calendar:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/calendar-signal ./signals/calendar/micro

resource:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/resource-signal ./signals/resource/micro

webhook:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/webhook-signal ./signals/webhook/micro

stream:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${STREAM}-signal ./signals/stream/builtin/${STREAM}/micro

# signal microservice docker images
artifact-image: artifact
	docker build -t $(IMAGE_PREFIX)artifact-signal:$(IMAGE_TAG) -f ./signals/artifact/micro/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)artifact-signal:$(IMAGE_TAG) ; fi

calendar-image: calendar
	docker build -t $(IMAGE_PREFIX)calendar-signal:$(IMAGE_TAG) -f ./signals/calendar/micro/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)calendar-signal:$(IMAGE_TAG) ; fi

resource-image: resource
	docker build -t $(IMAGE_PREFIX)resource-signal:$(IMAGE_TAG) -f ./signals/resource/micro/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)resource-signal:$(IMAGE_TAG) ; fi

webhook-image: webhook
	docker build -t $(IMAGE_PREFIX)webhook-signal:$(IMAGE_TAG) -f ./signals/webhook/micro/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)webhook-signal:$(IMAGE_TAG) ; fi

stream-image: stream
	docker build -t $(IMAGE_PREFIX)stream-$(STREAM)-signal:$(IMAGE_TAG) -f ./signals/stream/builtin/$(STREAM)/micro/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)stream-$(STREAM)-signal:$(IMAGE_TAG) ; fi

test:
	go test $(shell go list ./... | grep -v /vendor/) -race -short -v

coverage:
	go test -covermode=count -coverprofile=coverage.out $(shell go list ./... | grep -v /vendor/)
	go tool cover -func=coverage.out

clean:
	-rm -rf ${CURRENT_DIR}/dist
