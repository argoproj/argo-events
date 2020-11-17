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
IMAGE_TAG?=latest

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
all: sensor sensor-controller eventbus-controller eventsource-controller eventsource

all-images: sensor-image sensor-controller-image eventbus-controller-image eventsource-controller-image eventsource-image

all-controller-images: sensor-controller-image eventbus-controller-image eventsource-controller-image

.PHONY: all clean test

# EventSource controller
.PHONY: eventsource-controller
eventsource-controller: dist/eventsource-controller-linux-amd64

dist/eventsource-controller: GOARGS = GOOS= GOARCH=
dist/eventsource-controller-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/eventsource-controller-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/eventsource-controller-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/eventsource-controller-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/eventsource-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventsource-controller ./controllers/eventsource/cmd

dist/eventsource-controller-%:
	CGO_ENABLED=0 $(GOARGS) go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventsource-controller ./controllers/eventsource/cmd

eventsource-controller-image: dist/eventsource-controller-linux-amd64
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_PREFIX)eventsource-controller:$(IMAGE_TAG)  --target eventsource-controller -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)eventsource-controller:$(IMAGE_TAG) ; fi

# EventSource
.PHONY: eventsource
eventsource: dist/eventsource-linux-amd64

dist/eventsource: GOARGS = GOOS= GOARCH=
dist/eventsource-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/eventsource-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/eventsource-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/eventsource-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/eventsource:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventsource ./eventsources/cmd/main.go

dist/eventsource-%:
	CGO_ENABLED=0 $(GOARGS) go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventsource ./eventsources/cmd/main.go

eventsource-image: dist/eventsource-linux-amd64
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_PREFIX)eventsource:$(IMAGE_TAG)  --target eventsource -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)eventsource:$(IMAGE_TAG) ; fi

# Sensor controller
.PHONY: sensor-controller
sensor-controller: dist/sensor-controller-linux-amd64

dist/sensor-controller: GOARGS = GOOS= GOARCH=
dist/sensor-controller-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/sensor-controller-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/sensor-controller-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/sensor-controller-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/sensor-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor-controller ./controllers/sensor/cmd

dist/sensor-controller-%:
	CGO_ENABLED=0 $(GOARGS) go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor-controller ./controllers/sensor/cmd

sensor-controller-image: dist/sensor-controller-linux-amd64
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG)  --target sensor-controller -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) ; fi

# Sensor
.PHONY: sensor
sensor: dist/sensor-linux-amd64

dist/sensor: GOARGS = GOOS= GOARCH=
dist/sensor-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/sensor-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/sensor-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/sensor-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/sensor:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor ./sensors/cmd/main.go

dist/sensor-%:
	CGO_ENABLED=0 $(GOARGS) go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor ./sensors/cmd/main.go

sensor-image: dist/sensor-linux-amd64
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_PREFIX)sensor:$(IMAGE_TAG)  --target sensor -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)sensor:$(IMAGE_TAG) ; fi

# EventBus controller
.PHONY: eventbus-controller
eventbus-controller: dist/eventbus-controller-linux-amd64

dist/eventbus-controller: GOARGS = GOOS=linux GOARCH=amd64
dist/eventbus-controller-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/eventbus-controller-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/eventbus-controller-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/eventbus-controller-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/eventbus-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventbus-controller ./controllers/eventbus/cmd

dist/eventbus-controller-%:
	CGO_ENABLED=0 $(GOARGS) go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/eventbus-controller ./controllers/eventbus/cmd

eventbus-controller-image: dist/eventbus-controller-linux-amd64
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
	./hack/update-swagger.sh ${VERSION}

.PHONY: codegen
codegen:
	go mod vendor
	./hack/generate-proto.sh
	./hack/update-codegen.sh
	./hack/update-openapigen.sh
	$(MAKE) swagger
	./hack/update-api-docs.sh
	$(MAKE) manifests
	rm -rf ./vendor
	go mod tidy

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


