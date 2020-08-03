PACKAGE=github.com/argoproj/argo-events
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist

DOCKERFILE:=Dockerfile

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
DOCKER_PUSH?=false
IMAGE_NAMESPACE?=argoproj
IMAGE_TAG?=v0.17.0

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
all: sensor-linux sensor-controller-linux gateway-controller-linux eventbus-controller-linux eventsource-controller-linux eventsource-linux

all-images: sensor-image sensor-controller-image gateway-controller-image eventbus-controller-image eventsource-controller-image eventsource-image

all-controller-images: sensor-controller-image gateway-controller-image eventbus-controller-image eventsource-controller-image

.PHONY: all clean test

# EventSource
eventsource:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventsource ./eventsources/cmd/main.go

eventsource-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make eventsource

eventsource-image:
	make eventsource-linux
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_PREFIX)eventsource:$(IMAGE_TAG)  --target eventsource -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)eventsource:$(IMAGE_TAG) ; fi

# EventSource controller
eventsource-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventsource-controller ./controllers/eventsource/cmd

eventsource-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make eventsource-controller

eventsource-controller-image:
	make eventsource-controller-linux
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_PREFIX)eventsource-controller:$(IMAGE_TAG)  --target eventsource-controller -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)eventsource-controller:$(IMAGE_TAG) ; fi

# Sensor
sensor:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor ./sensors/cmd/main.go

sensor-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sensor

sensor-image:
	make sensor-linux
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_PREFIX)sensor:$(IMAGE_TAG)  --target sensor -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)sensor:$(IMAGE_TAG) ; fi

# Sensor controller
sensor-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor-controller ./controllers/sensor/cmd

sensor-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sensor-controller

sensor-controller-image:
	make sensor-controller-linux
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG)  --target sensor-controller -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) ; fi

# Gateway controller
gateway-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-controller ./controllers/gateway/cmd

gateway-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-controller

gateway-controller-image:
	make gateway-controller-linux
	docker build -t $(IMAGE_PREFIX)gateway-controller:$(IMAGE_TAG) -f ./controllers/gateway/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gateway-controller:$(IMAGE_TAG) ; fi

# EventBus controller
eventbus-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventbus-controller ./controllers/eventbus/cmd

eventbus-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make eventbus-controller

eventbus-controller-image:
	make eventbus-controller-linux
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_PREFIX)eventbus-controller:$(IMAGE_TAG)  --target eventbus-controller -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)eventbus-controller:$(IMAGE_TAG) ; fi

test:
	go test $(shell go list ./... | grep -v /vendor/ | grep -v /test/e2e/) -race -short -v

coverage:
	go test -covermode=count -coverprofile=profile.cov $(shell go list ./... | grep -v /vendor/ | grep -v /test/e2e/)
	go tool cover -func=profile.cov

clean:
	-rm -rf ${CURRENT_DIR}/dist

.PHONY: crds
crds:
	./hack/crdgen.sh

.PHONY: manifests
manifests: crds
	kustomize build manifests/cluster-install > manifests/install.yaml
	kustomize build manifests/namespace-install > manifests/namespace-install.yaml

.PHONY: swagger
swagger:
	go run ./hack/gen-openapi-spec/main.go ${VERSION} > ${CURRENT_DIR}/api/openapi-spec/swagger.json

.PHONY: codegen
codegen:
	go mod vendor
	./hack/generate-proto.sh
	./hack/update-codegen.sh
	./hack/update-openapigen.sh
	$(MAKE) swagger
	./hack/update-api-docs.sh
	./hack/update-mocks.sh
	rm -rf ./vendor
	go mod tidy
	$(MAKE) manifests

.PHONY: start
start:
	kustomize build --load_restrictor=none test/manifests > /tmp/argo-events.yaml
	kubectl apply -f test/manifests/argo-events-ns.yaml
	kubectl -n argo-events apply -l app.kubernetes.io/part-of=argo-events --prune --force -f /tmp/argo-events.yaml
	kubectl -n argo-events wait --for=condition=Ready --timeout 60s pod --all
	kubens argo-events

$(GOPATH)/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.26.0

.PHONY: lint
lint: $(GOPATH)/bin/golangci-lint
	go mod tidy
	golangci-lint run --fix --verbose --concurrency 4 --timeout 5m


