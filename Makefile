PACKAGE=github.com/argoproj/argo-events
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist

DOCKERFILE:=Dockerfile

BINARY_NAME:=argo-events

BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_BRANCH=$(shell git rev-parse --symbolic-full-name --verify --quiet --abbrev-ref HEAD)
GIT_TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)
EXECUTABLES = curl docker gzip go

#  docker image publishing options
DOCKER_PUSH?=false
IMAGE_NAMESPACE?=quay.io/argoproj
VERSION?=latest
BASE_VERSION:=latest

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.gitTreeState=${GIT_TREE_STATE}

ifeq (${DOCKER_PUSH},true)
PUSH_OPTION="--push"
ifndef IMAGE_NAMESPACE
$(error IMAGE_NAMESPACE must be set to push images (e.g. IMAGE_NAMESPACE=quay.io/argoproj))
endif
endif

ifneq (${GIT_TAG},)
VERSION=$(GIT_TAG)
override LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif

K3D ?= $(shell [ "`command -v kubectl`" != '' ] && [ "`command -v k3d`" != '' ] && [[ "`kubectl config current-context`" =~ k3d-* ]] && echo true || echo false)

# Check that the needed executables are available, else exit before the build
K := $(foreach exec,$(EXECUTABLES), $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

.PHONY: build image clean test

# build
.PHONY: build
build: dist/$(BINARY_NAME)-linux-amd64.gz dist/$(BINARY_NAME)-linux-arm64.gz dist/$(BINARY_NAME)-linux-arm.gz dist/$(BINARY_NAME)-linux-ppc64le.gz dist/$(BINARY_NAME)-linux-s390x.gz

dist/$(BINARY_NAME)-%.gz: dist/$(BINARY_NAME)-%
	@[ -e dist/$(BINARY_NAME)-$*.gz ] || gzip -k dist/$(BINARY_NAME)-$*

dist/$(BINARY_NAME): GOARGS = GOOS= GOARCH=
dist/$(BINARY_NAME)-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/$(BINARY_NAME)-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/$(BINARY_NAME)-linux-arm: GOARGS = GOOS=linux GOARCH=arm
dist/$(BINARY_NAME)-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/$(BINARY_NAME)-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/$(BINARY_NAME):
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/$(BINARY_NAME) ./cmd

dist/$(BINARY_NAME)-%:
	CGO_ENABLED=0 $(GOARGS) go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/$(BINARY_NAME)-$* ./cmd

.PHONY: image
image: clean dist/$(BINARY_NAME)-linux-amd64
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_NAMESPACE)/$(BINARY_NAME):$(VERSION)  --target $(BINARY_NAME) -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ]; then docker push $(IMAGE_NAMESPACE)/$(BINARY_NAME):$(VERSION); fi
ifeq ($(K3D),true)
	k3d image import $(IMAGE_NAMESPACE)/$(BINARY_NAME):$(VERSION)
endif

image-linux-%: dist/$(BINARY_NAME)-linux-%
	DOCKER_BUILDKIT=1 docker build --build-arg "ARCH=$*" -t $(IMAGE_NAMESPACE)/$(BINARY_NAME):$(VERSION)-linux-$* --platform "linux/$*" --target $(BINARY_NAME) -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ]; then docker push $(IMAGE_NAMESPACE)/$(BINARY_NAME):$(VERSION)-linux-$*; fi

image-multi: set-qemu dist/$(BINARY_NAME)-linux-arm64.gz dist/$(BINARY_NAME)-linux-amd64.gz
	docker buildx build --sbom=false --provenance=false --tag $(IMAGE_NAMESPACE)/$(BINARY_NAME):$(VERSION) --target $(BINARY_NAME) --platform linux/amd64,linux/arm64 --file ./Dockerfile ${PUSH_OPTION} .

set-qemu:
	docker pull tonistiigi/binfmt:latest
	docker run --rm --privileged tonistiigi/binfmt:latest --install amd64,arm64

test:
	go test $(shell go list ./... | grep -v /vendor/ | grep -v /test/e2e/) -race -short -v

test-functional:
ifeq ($(EventBusDriver),kafka)
	kubectl -n argo-events apply -k test/manifests/kafka
	kubectl -n argo-events wait -l statefulset.kubernetes.io/pod-name=kafka-0 --for=condition=ready pod --timeout=60s
endif
	go test -v -timeout 20m -count 1 --tags functional -p 1 ./test/e2e
ifeq ($(EventBusDriver),kafka)
	kubectl -n argo-events delete -k test/manifests/kafka
endif

# to run just one of the functional e2e tests by name (i.e. 'make TestMetricsWithWebhook'):
Test%:
	go test -v -timeout 10m -count 1 --tags functional -p 1 ./test/e2e  -run='.*/$*'


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
	kubectl kustomize manifests/cluster-install > manifests/install.yaml
	kubectl kustomize manifests/namespace-install > manifests/namespace-install.yaml
	kubectl kustomize manifests/extensions/validating-webhook > manifests/install-validating-webhook.yaml

.PHONY: swagger
swagger:
	./hack/update-swagger.sh ${VERSION}
	$(MAKE) api/jsonschema/schema.json

api/jsonschema/schema.json: api/openapi-spec/swagger.json hack/jsonschema/main.go
	go run ./hack/jsonschema

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
start: image
	kubectl apply -f test/manifests/argo-events-ns.yaml
	kubectl kustomize test/manifests | sed 's@quay.io/argoproj/@$(IMAGE_NAMESPACE)/@' | sed 's/:$(BASE_VERSION)/:$(VERSION)/' | kubectl -n argo-events apply -l app.kubernetes.io/part-of=argo-events --prune=false --force -f -
	kubectl -n argo-events wait --for=condition=Ready --timeout 60s pod --all

$(GOPATH)/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.49.0

.PHONY: lint
lint: $(GOPATH)/bin/golangci-lint
	go mod tidy
	golangci-lint run --fix --verbose --concurrency 4 --timeout 10m

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
	cat manifests/base/kustomization.yaml | sed 's/newTag: .*/newTag: $(VERSION)/' | sed 's@value: quay.io/argoproj/argo-events:.*@value: quay.io/argoproj/argo-events:$(VERSION)@' > /tmp/base_kustomization.yaml
	mv /tmp/base_kustomization.yaml manifests/base/kustomization.yaml
	cat manifests/extensions/validating-webhook/kustomization.yaml | sed 's/newTag: .*/newTag: $(VERSION)/' > /tmp/wh_kustomization.yaml
	mv /tmp/wh_kustomization.yaml manifests/extensions/validating-webhook/kustomization.yaml
	cat Makefile | sed 's/^VERSION?=.*/VERSION?=$(VERSION)/' | sed 's/^BASE_VERSION:=.*/BASE_VERSION:=$(VERSION)/' > /tmp/ae_makefile
	mv /tmp/ae_makefile Makefile

.PHONY: checksums
checksums:
	sha256sum ./dist/$(BINARY_NAME)-*.gz | awk -F './dist/' '{print $$1 $$2}' > ./dist/$(BINARY_NAME)-checksums.txt
