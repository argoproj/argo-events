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

#  docker image publishing options
DOCKER_PUSH?=true
IMAGE_NAMESPACE?=argoproj
IMAGE_TAG?=v0.16.0
BUILD_BINARY?=true

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

# Build the project images
.DELETE_ON_ERROR:
all: sensor-linux sensor-controller-linux gateway-controller-linux gateway-client-linux gateway-server-linux eventbus-controller-linux

all-images: sensor-image sensor-controller-image gateway-controller-image gateway-client-image gateway-server-image eventbus-controller-image

all-controller-images: sensor-controller-image gateway-controller-image eventbus-controller-image

.PHONY: all clean test

# Sensor
sensor:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor ./sensors/cmd/client.go

sensor-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sensor

sensor-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make sensor-linux; fi
	docker build -t $(IMAGE_PREFIX)sensor:$(IMAGE_TAG) -f ./sensors/cmd/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)sensor:$(IMAGE_TAG) ; fi

# Sensor controller
sensor-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor-controller ./controllers/sensor/cmd

sensor-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sensor-controller

sensor-controller-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make sensor-controller-linux; fi
	docker build -t $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) -f ./controllers/sensor/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) ; fi

# Gateway controller
gateway-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-controller ./controllers/gateway/cmd

gateway-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-controller

gateway-controller-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make gateway-controller-linux; fi
	docker build -t $(IMAGE_PREFIX)gateway-controller:$(IMAGE_TAG) -f ./controllers/gateway/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gateway-controller:$(IMAGE_TAG) ; fi

# EventBus controller
eventbus-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventbus-controller ./controllers/eventbus/cmd

eventbus-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make eventbus-controller

eventbus-controller-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make eventbus-controller-linux; fi
	docker build -t $(IMAGE_PREFIX)eventbus-controller:$(IMAGE_TAG) -f ./controllers/eventbus/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)eventbus-controller:$(IMAGE_TAG) ; fi

# Gateway client binary
gateway-client:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-client ./gateways/client

gateway-client-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-client

gateway-client-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make gateway-client-linux; fi
	docker build -t $(IMAGE_PREFIX)gateway-client:$(IMAGE_TAG) -f ./gateways/client/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gateway-client:$(IMAGE_TAG) ; fi


# gateway binary
gateway-server:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-server ./gateways/server/cmd/

gateway-server-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-server

gateway-server-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make gateway-server-linux; fi
	docker build -t $(IMAGE_PREFIX)gateway-server:$(IMAGE_TAG) -f ./gateways/server/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gateway-server:$(IMAGE_TAG) ; fi

test:
	go test $(shell go list ./... | grep -v /vendor/ | grep -v /test/e2e/) -race -short -v

coverage:
	go test -covermode=count -coverprofile=profile.cov $(shell go list ./... | grep -v /vendor/ | grep -v /test/e2e/)
	go tool cover -func=profile.cov

clean:
	-rm -rf ${CURRENT_DIR}/dist

.PHONY: codegen
codegen:
	go mod vendor
	./hack/update-codegen.sh
	./hack/update-openapigen.sh
	go run ./hack/gen-openapi-spec/main.go ${VERSION} > ${CURRENT_DIR}/api/openapi-spec/swagger.json
	./hack/update-api-docs.sh
	rm -rf ./vendor
	go mod tidy

.PHONY: e2e
e2e:
	./hack/e2e/run-e2e.sh

.PHONY: kind-e2e
kind-e2e:
	./hack/e2e/kind-run-e2e.sh

.PHONY: build-e2e-images
build-e2e-images: sensor-controller-image gateway-controller-image gateway-client-image gateway-server-image

.PHONY: lint
lint:
	golangci-lint run --fix

