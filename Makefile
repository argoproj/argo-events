PACKAGE=github.com/argoproj/argo-events
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist

DOCKERFILE:=Dockerfile

BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_BRANCH=$(shell git rev-parse --symbolic-full-name --verify --quiet --abbrev-ref HEAD)
GIT_TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

#  docker image publishing options
DOCKER_PUSH?=false
IMAGE_NAMESPACE?=argoproj
VERSION?=v1.3.0-rc3

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.gitTreeState=${GIT_TREE_STATE}

ifeq (${DOCKER_PUSH},true)
ifndef IMAGE_NAMESPACE
$(error IMAGE_NAMESPACE must be set to push images (e.g. IMAGE_NAMESPACE=argoproj))
endif
endif

ifneq (${GIT_TAG},)
VERSION=$(GIT_TAG)
override LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif

# Build the project images
.DELETE_ON_ERROR:
all: sensor sensor-controller eventbus-controller eventsource-controller eventsource events-webhook

all-images: sensor-image sensor-controller-image eventbus-controller-image eventsource-controller-image eventsource-image events-webhook-image

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
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_NAMESPACE)/eventsource-controller:$(VERSION)  --target eventsource-controller -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ]; then docker push $(IMAGE_NAMESPACE)/eventsource-controller:$(VERSION); fi

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
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_NAMESPACE)/eventsource:$(VERSION)  --target eventsource -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ]; then docker push $(IMAGE_NAMESPACE)/eventsource:$(VERSION); fi

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
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_NAMESPACE)/sensor-controller:$(VERSION)  --target sensor-controller -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ]; then docker push $(IMAGE_NAMESPACE)/sensor-controller:$(VERSION); fi

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
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_NAMESPACE)/sensor:$(VERSION)  --target sensor -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ]; then docker push $(IMAGE_NAMESPACE)/sensor:$(VERSION); fi

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
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_NAMESPACE)/eventbus-controller:$(VERSION)  --target eventbus-controller -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ]; then docker push $(IMAGE_NAMESPACE)/eventbus-controller:$(VERSION); fi

# Webhook
.PHONY: events-webhook
events-webhook: dist/events-webhook-linux-amd64

dist/events-webhook: GOARGS = GOOS= GOARCH=
dist/events-webhook-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/events-webhook-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/events-webhook-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/events-webhook-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/events-webhook:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/events-webhook ./webhook/cmd/main.go

dist/events-webhook-%:
	CGO_ENABLED=0 $(GOARGS) go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/events-webhook ./webhook/cmd/main.go

events-webhook-image: dist/events-webhook-linux-amd64
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_NAMESPACE)/events-webhook:$(VERSION)  --target events-webhook -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ]; then docker push $(IMAGE_NAMESPACE)/events-webhook:$(VERSION); fi

test:
	go test $(shell go list ./... | grep -v /vendor/ | grep -v /test/e2e/) -race -short -v

test-functional:
	go test -v -timeout 10m -count 1 --tags functional -p 1 ./test/e2e

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
	kustomize build manifests/extensions/validating-webhook > manifests/install-validating-webhook.yaml

.PHONY: swagger
swagger:
	./hack/update-swagger.sh ${VERSION}

.PHONY: codegen
codegen:
	./hack/generate-proto.sh
	./hack/update-codegen.sh
	./hack/update-openapigen.sh
	$(MAKE) swagger
	./hack/update-api-docs.sh
	$(MAKE) manifests
	rm -rf ./vendor
	go mod tidy

go-diagrams/diagram.dot: ./hack/diagram/main.go
	rm -Rf go-diagrams
	go run ./hack/diagram

docs/assets/diagram.png: go-diagrams/diagram.dot
	cd go-diagrams && dot -Tpng diagram.dot -o ../docs/assets/diagram.png

.PHONY: start
start: all-images
	kubectl apply -f test/manifests/argo-events-ns.yaml
	kustomize build test/manifests | sed 's@argoproj/@$(IMAGE_NAMESPACE)/@' | sed 's/:latest/:$(VERSION)/' | kubectl -n argo-events apply -l app.kubernetes.io/part-of=argo-events --prune --force -f -
	kubectl -n argo-events wait --for=condition=Ready --timeout 60s pod --all

$(GOPATH)/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.26.0

.PHONY: lint
lint: $(GOPATH)/bin/golangci-lint
	go mod tidy
	golangci-lint run --fix --verbose --concurrency 4 --timeout 5m

.PHONY: quay-release
quay-release: eventbus-controller-image sensor-controller-image sensor-image eventsource-image eventsource-controller-image
	docker tag $(IMAGE_NAMESPACE)/eventbus-controller:$(VERSION) quay.io/$(IMAGE_NAMESPACE)/eventbus-controller:$(VERSION)
	docker push quay.io/$(IMAGE_NAMESPACE)/eventbus-controller:$(VERSION)

	docker tag $(IMAGE_NAMESPACE)/sensor:$(VERSION) quay.io/$(IMAGE_NAMESPACE)/sensor:$(VERSION)
	docker push quay.io/$(IMAGE_NAMESPACE)/sensor:$(VERSION)

	docker tag $(IMAGE_NAMESPACE)/sensor-controller:$(VERSION) quay.io/$(IMAGE_NAMESPACE)/sensor-controller:$(VERSION)
	docker push quay.io/$(IMAGE_NAMESPACE)/sensor-controller:$(VERSION)

	docker tag $(IMAGE_NAMESPACE)/eventsource-controller:$(VERSION) quay.io/$(IMAGE_NAMESPACE)/eventsource-controller:$(VERSION)
	docker push quay.io/$(IMAGE_NAMESPACE)/eventsource-controller:$(VERSION)

	docker tag $(IMAGE_NAMESPACE)/eventsource:$(VERSION) quay.io/$(IMAGE_NAMESPACE)/eventsource:$(VERSION)
	docker push quay.io/$(IMAGE_NAMESPACE)/eventsource:$(VERSION)

	docker tag $(IMAGE_NAMESPACE)/events-webhook:$(VERSION) quay.io/$(IMAGE_NAMESPACE)/events-webhook:$(VERSION)
	docker push quay.io/$(IMAGE_NAMESPACE)/events-webhook:$(VERSION)

# release - targets only available on release branch
ifneq ($(findstring release,$(GIT_BRANCH)),)

.PHONY: prepare-release
prepare-release: check-version-warning clean update-manifests-version codegen
	git status
	@git diff --quiet || echo "\n\nPlease run 'git diff' to confirm the file changes are correct.\n"

.PHONY: release
release: check-version-warning
	@echo "\n1. Make sure you have run 'VERSION=$(VERSION) make prepare-release', and confirmed all the changes are expected."
	@echo "\n2. Run following commands to commit the changes to the release branch, add give a tag.\n"
	@echo "git commit -am \"Update manifests to $(VERSION)\""
	@echo "git push {your-remote}\n"
	@echo "git tag -a $(VERSION) -m $(VERSION)"
	@echo "git push {your-remote} $(VERSION)\n"

endif

.PHONY: check-version-warning
check-version-warning:
	@if [[ ! "$(VERSION)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+.*$  ]]; then echo -n "It looks like you're not using a version format like 'v1.2.3', or 'v1.2.3-rc2', that version format is required for our releases. Do you wish to continue anyway? [y/N]" && read ans && [ $${ans:-N} = y ]; fi

.PHONY: update-manifests-version
update-manifests-version:
	cat manifests/base/kustomization.yaml | sed 's/newTag: .*/newTag: $(VERSION)/' | sed 's@value: argoproj/eventsource:.*@value: argoproj/eventsource:$(VERSION)@' | sed 's@value: argoproj/sensor:.*@value: argoproj/sensor:$(VERSION)@' > /tmp/base_kustomization.yaml
	mv /tmp/base_kustomization.yaml manifests/base/kustomization.yaml
	cat manifests/extensions/validating-webhook/kustomization.yaml | sed 's/newTag: .*/newTag: $(VERSION)/' > /tmp/wh_kustomization.yaml
	mv /tmp/wh_kustomization.yaml manifests/extensions/validating-webhook/kustomization.yaml
	cat Makefile | sed 's/^VERSION?=.*/VERSION?=$(VERSION)/' > /tmp/ae_makefile
	mv /tmp/ae_makefile Makefile

