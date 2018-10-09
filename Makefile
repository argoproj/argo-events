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
DOCKER_PUSH=true
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

# Build the project images
.DELETE_ON_ERROR:
all: sensor-linux sensor-controller-linux gateway-controller-linux gateway-http-transformer-linux webhook-linux calendar-linux resource-linux artifact-linux nats-linux kafka-linux amqp-linux mqtt-linux gateway-processor-grpc-client-linux calendar-grpc-linux gateway-processor-http-client-linux calendar-http-linux

all-images: sensor-image sensor-controller-image gateway-controller-image gateway-http-transformer-image webhook-image calendar-image resource-image artifact-image nats-image kafka-image amqp-image mqtt-image gateway-processor-grpc-client-image calendar-grpc-image gateway-processor-http-client-image calendar-http-image

all-controller-images: sensor-controller-image gateway-controller-image

all-core-gateway-images: webhook-image calendar-image artifact-image nats-image kafka-image amqp-image mqtt-image resource-image

.PHONY: all clean test

# Sensor
sensor:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor ./controllers/sensor/cmd/

sensor-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sensor

sensor-image: sensor-linux
	 docker build -t $(IMAGE_PREFIX)sensor:$(IMAGE_TAG) -f ./controllers/sensor/cmd/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)sensor:$(IMAGE_TAG) ; fi


# Sensor controller
sensor-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor-controller ./cmd/controllers/sensor

sensor-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sensor-controller

sensor-controller-image: sensor-controller-linux
	 docker build -t $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) -f ./controllers/sensor/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) ; fi


# Gateway controller
gateway-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-controller ./cmd/controllers/gateway/main.go

gateway-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-controller

gateway-controller-image: gateway-controller-linux
	 docker build -t $(IMAGE_PREFIX)gateway-controller:$(IMAGE_TAG) -f ./controllers/gateway/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gateway-controller:$(IMAGE_TAG) ; fi


# Gateway transformers
gateway-http-transformer:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-http-transformer ./cmd/controllers/gateway/transform/http/main.go

gateway-http-transformer-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-http-transformer

gateway-http-transformer-image: gateway-http-transformer-linux
	 docker build -t $(IMAGE_PREFIX)gateway-http-transformer:$(IMAGE_TAG) -f ./controllers/gateway/transform/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gateway-http-transformer:$(IMAGE_TAG) ; fi

# gateway binaries
webhook:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/webhook-gateway ./gateways/core/webhook/

webhook-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make webhook

webhook-image: webhook-linux
	 docker build -t $(IMAGE_PREFIX)webhook-gateway:$(IMAGE_TAG) -f ./gateways/core/webhook/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)webhook-gateway:$(IMAGE_TAG) ; fi


calendar:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/calendar-gateway ./gateways/core/calendar/

calendar-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make calendar

calendar-image: calendar-linux
	 docker build -t $(IMAGE_PREFIX)calendar-gateway:$(IMAGE_TAG) -f ./gateways/core/calendar/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)calendar-gateway:$(IMAGE_TAG) ; fi


resource:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/resource-gateway ./gateways/core/resource/

resource-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make resource

resource-image: resource-linux
	 docker build -t $(IMAGE_PREFIX)resource-gateway:$(IMAGE_TAG) -f ./gateways/core/resource/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)resource-gateway:$(IMAGE_TAG) ; fi



artifact:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/artifact-gateway ./gateways/core/artifact/

artifact-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make artifact

artifact-image: artifact-linux
	 docker build -t $(IMAGE_PREFIX)artifact-gateway:$(IMAGE_TAG) -f ./gateways/core/artifact/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)artifact-gateway:$(IMAGE_TAG) ; fi


file-watcher:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/file-watcher-gateway ./gateways/core/file-watcher/

file-watcher-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make file-watcher

file-watcher-image: file-watcher-linux
	 docker build -t $(IMAGE_PREFIX)file-watcher-gateway:$(IMAGE_TAG) -f ./gateways/core/file-watcher/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)file-watcher-gateway:$(IMAGE_TAG) ; fi



nats:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/nats-gateway ./gateways/core/stream/nats/

nats-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make nats

nats-image: nats-linux
	 docker build -t $(IMAGE_PREFIX)nats-gateway:$(IMAGE_TAG) -f ./gateways/core/stream/nats/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)nats-gateway:$(IMAGE_TAG) ; fi


kafka:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/kafka-gateway ./gateways/core/stream/kafka/

kafka-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make kafka

kafka-image: kafka-linux
	 docker build -t $(IMAGE_PREFIX)kafka-gateway:$(IMAGE_TAG) -f ./gateways/core/stream/kafka/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)kafka-gateway:$(IMAGE_TAG) ; fi


amqp:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/amqp-gateway ./gateways/core/stream/amqp/

amqp-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make amqp

amqp-image: amqp-linux
	 docker build -t $(IMAGE_PREFIX)amqp-gateway:$(IMAGE_TAG) -f ./gateways/core/stream/amqp/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)amqp-gateway:$(IMAGE_TAG) ; fi


mqtt:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/mqtt-gateway ./gateways/core/stream/mqtt/

mqtt-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make mqtt

mqtt-image: mqtt-linux
	 docker build -t $(IMAGE_PREFIX)mqtt-gateway:$(IMAGE_TAG) -f ./gateways/core/stream/mqtt/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)mqtt-gateway:$(IMAGE_TAG) ; fi


# gRPC gateway binary
grpc-gateway-images:  gateway-processor-grpc-client-image calendar-grpc-image

# Gateway processor client
gateway-processor-grpc-client:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-processor-client ./gateways/grpc/gateway.go

gateway-processor-grpc-client-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-processor-grpc-client

gateway-processor-grpc-client-image: gateway-processor-grpc-client-linux
	 docker build -t $(IMAGE_PREFIX)gateway-processor-grpc-client:$(IMAGE_TAG) -f ./gateways/grpc/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gateway-processor-grpc-client:$(IMAGE_TAG) ; fi

# gRPC gateway server binaries
calendar-grpc:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/calendar-grpc-gateway ./gateways/grpc/calendar/calendar.go

calendar-grpc-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make calendar-grpc

calendar-grpc-image: calendar-grpc-linux
	 docker build -t $(IMAGE_PREFIX)calendar-grpc-gateway:$(IMAGE_TAG) -f ./gateways/grpc/calendar/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)calendar-grpc-gateway:$(IMAGE_TAG) ; fi


# HTTP gateway client binary
http-gateway-images: gateway-processor-http-client-image calendar-http-image

# Gateway processor client
gateway-processor-http-client:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-processor-http-client ./gateways/rest/gateway.go

gateway-processor-http-client-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-processor-http-client

gateway-processor-http-client-image: gateway-processor-http-client-linux
	 docker build -t $(IMAGE_PREFIX)gateway-processor-http-client:$(IMAGE_TAG) -f ./gateways/rest/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gateway-processor-http-client:$(IMAGE_TAG) ; fi

# HTTP gateway server binaries
calendar-http:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/calendar-http ./gateways/rest/calendar/calendar.go

calendar-http-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make calendar-http

calendar-http-image: calendar-http-linux
	 docker build -t $(IMAGE_PREFIX)calendar-http-gateway:$(IMAGE_TAG) -f ./gateways/rest/calendar/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)calendar-http-gateway:$(IMAGE_TAG) ; fi

test:
	go test $(shell go list ./... | grep -v /vendor/) -race -short -v

coverage:
	go test -covermode=count -coverprofile=coverage.out $(shell go list ./... | grep -v /vendor/)
	go tool cover -func=coverage.out

clean:
	-rm -rf ${CURRENT_DIR}/dist

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