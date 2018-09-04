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

# sudo docker image publishing options
DOCKER_PUSH=true
IMAGE_NAMESPACE=metalgearsolid
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
all: sensor-linux sensor-controller-linux gateway-controller-linux gateway-transformer-linux webhook-linux calendar-linux artifact-linux nats-linux kafka-linux amqp-linux mqtt-linux

all-images: sensor-image sensor-controller-image gateway-controller-image gateway-transformer-image webhook-image calendar-image artifact-image nats-image kafka-image amqp-image mqtt-image

all-controller-images: sensor-controller-image gateway-controller-image

all-gateway-images: webhook-image calendar-image artifact-image nats-image kafka-image amqp-image mqtt-image

.PHONY: all clean test

# Sensor
sensor:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor ./controllers/sensor/cmd/

sensor-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sensor

sensor-image: sensor-linux
	sudo docker build -t $(IMAGE_PREFIX)sensor:$(IMAGE_TAG) -f ./controllers/sensor/cmd/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)sensor:$(IMAGE_TAG) ; fi


# Sensor controller
sensor-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/sensor-controller ./cmd/controllers/sensor

sensor-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sensor-controller

sensor-controller-image: sensor-controller-linux
	sudo docker build -t $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) -f ./controllers/sensor/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)sensor-controller:$(IMAGE_TAG) ; fi


# Gateway controller
gateway-controller:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-controller ./cmd/controllers/gateway/main.go

gateway-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-controller

gateway-controller-image: gateway-controller-linux
	sudo docker build -t $(IMAGE_PREFIX)gateway-controller:$(IMAGE_TAG) -f ./controllers/gateway/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)gateway-controller:$(IMAGE_TAG) ; fi


# Gateway transformer
gateway-transformer:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-transformer ./cmd/controllers/gateway/transform/main.go

gateway-transformer-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-transformer

gateway-transformer-image: gateway-transformer-linux
	sudo docker build -t $(IMAGE_PREFIX)gateway-transformer:$(IMAGE_TAG) -f ./controllers/gateway/transform/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)gateway-transformer:$(IMAGE_TAG) ; fi


# Gateway processor client
gateway-processor-client:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-processor-client ./gateways/gateway.go

gateway-processor-client-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-processor-client

gateway-processor-client-image: gateway-processor-client-linux
	sudo docker build -t $(IMAGE_PREFIX)gateway-processor-client:$(IMAGE_TAG) -f ./gateways/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)gateway-processor-client:$(IMAGE_TAG) ; fi



# gateway binaries
webhook:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/webhook-gateway ./gateways/core/webhook/

webhook-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make webhook

webhook-image: webhook-linux
	sudo docker build -t $(IMAGE_PREFIX)webhook-gateway:$(IMAGE_TAG) -f ./gateways/core/webhook/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)webhook-gateway:$(IMAGE_TAG) ; fi


calendar:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/calendar-gateway ./gateways/core/calendar/

calendar-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make calendar

calendar-image: calendar-linux
	sudo docker build -t $(IMAGE_PREFIX)calendar-gateway:$(IMAGE_TAG) -f ./gateways/core/calendar/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)calendar-gateway:$(IMAGE_TAG) ; fi


artifact:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/artifact-gateway ./gateways/core/artifact/

artifact-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make artifact

artifact-image: artifact-linux
	sudo docker build -t $(IMAGE_PREFIX)artifact-gateway:$(IMAGE_TAG) -f ./gateways/core/artifact/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)artifact-gateway:$(IMAGE_TAG) ; fi


nats:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/nats-gateway ./gateways/core/stream/nats/

nats-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make nats

nats-image: nats-linux
	sudo docker build -t $(IMAGE_PREFIX)nats-gateway:$(IMAGE_TAG) -f ./gateways/core/stream/nats/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)nats-gateway:$(IMAGE_TAG) ; fi


kafka:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/kafka-gateway ./gateways/core/stream/kafka/

kafka-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make kafka

kafka-image: kafka-linux
	sudo docker build -t $(IMAGE_PREFIX)kafka-gateway:$(IMAGE_TAG) -f ./gateways/core/stream/kafka/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)kafka-gateway:$(IMAGE_TAG) ; fi


amqp:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/amqp-gateway ./gateways/core/stream/amqp/

amqp-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make amqp

amqp-image: amqp-linux
	sudo docker build -t $(IMAGE_PREFIX)amqp-gateway:$(IMAGE_TAG) -f ./gateways/core/stream/amqp/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)amqp-gateway:$(IMAGE_TAG) ; fi


mqtt:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/mqtt-gateway ./gateways/core/stream/mqtt/

mqtt-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make mqtt

mqtt-image: mqtt-linux
	sudo docker build -t $(IMAGE_PREFIX)mqtt-gateway:$(IMAGE_TAG) -f ./gateways/core/stream/mqtt/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then sudo docker push $(IMAGE_PREFIX)mqtt-gateway:$(IMAGE_TAG) ; fi


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