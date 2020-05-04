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
IMAGE_TAG?=v0.14.0
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
all: sensor-linux sensor-controller-linux gateway-controller-linux gateway-client-linux webhook-linux calendar-linux resource-linux minio-linux file-linux nats-linux kafka-linux amqp-linux mqtt-linux storage-grid-linux github-linux hdfs-linux gitlab-linux sns-linux sqs-linux pubsub-linux slack-linux nsq-linux redis-linux emitter-linux stripe-linux azure-events-hub-linux

all-images: sensor-image sensor-controller-image gateway-controller-image gateway-client-image webhook-image calendar-image resource-image minio-image file-image nats-image kafka-image amqp-image mqtt-image storage-grid-image github-image gitlab-image sns-image pubsub-image hdfs-image sqs-image slack-image nsq-image redis-image emitter-image stripe-image azure-events-hub-image

all-controller-images: sensor-controller-image gateway-controller-image

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


# Gateway client binary
gateway-client:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gateway-client ./gateways/client

gateway-client-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gateway-client

gateway-client-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make gateway-client-linux; fi
	docker build -t $(IMAGE_PREFIX)gateway-client:$(IMAGE_TAG) -f ./gateways/client/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gateway-client:$(IMAGE_TAG) ; fi


# gateway binaries
webhook:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/webhook-gateway ./gateways/server/webhook/cmd/

webhook-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make webhook

webhook-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make webhook-linux; fi
	docker build -t $(IMAGE_PREFIX)webhook-gateway:$(IMAGE_TAG) -f ./gateways/server/webhook/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)webhook-gateway:$(IMAGE_TAG) ; fi


calendar:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/calendar-gateway ./gateways/server/calendar/cmd/

calendar-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make calendar

calendar-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make calendar-linux; fi
	docker build -t $(IMAGE_PREFIX)calendar-gateway:$(IMAGE_TAG) -f ./gateways/server/calendar/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)calendar-gateway:$(IMAGE_TAG) ; fi


resource:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/resource-gateway ./gateways/server/resource/cmd

resource-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make resource

resource-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make resource-linux; fi
	docker build -t $(IMAGE_PREFIX)resource-gateway:$(IMAGE_TAG) -f ./gateways/server/resource/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)resource-gateway:$(IMAGE_TAG) ; fi


minio:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/minio-gateway ./gateways/server/minio/cmd

minio-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make minio

minio-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make minio-linux; fi
	docker build -t $(IMAGE_PREFIX)artifact-gateway:$(IMAGE_TAG) -f ./gateways/server/minio/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)artifact-gateway:$(IMAGE_TAG) ; fi


file:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/file-gateway ./gateways/server/file/cmd

file-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make file

file-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make file-linux; fi
	docker build -t $(IMAGE_PREFIX)file-gateway:$(IMAGE_TAG) -f ./gateways/server/file/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)file-gateway:$(IMAGE_TAG) ; fi


#Stream gateways
nats:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/nats-gateway ./gateways/server/nats/cmd

nats-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make nats

nats-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make nats-linux; fi
	docker build -t $(IMAGE_PREFIX)nats-gateway:$(IMAGE_TAG) -f ./gateways/server/nats/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)nats-gateway:$(IMAGE_TAG) ; fi


kafka:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/kafka-gateway ./gateways/server/kafka/cmd

kafka-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make kafka

kafka-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make kafka-linux; fi
	docker build -t $(IMAGE_PREFIX)kafka-gateway:$(IMAGE_TAG) -f ./gateways/server/kafka/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)kafka-gateway:$(IMAGE_TAG) ; fi


amqp:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/amqp-gateway ./gateways/server/amqp/cmd

amqp-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make amqp

amqp-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make amqp-linux; fi
	docker build -t $(IMAGE_PREFIX)amqp-gateway:$(IMAGE_TAG) -f ./gateways/server/amqp/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)amqp-gateway:$(IMAGE_TAG) ; fi


mqtt:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/mqtt-gateway ./gateways/server/mqtt/cmd

mqtt-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make mqtt

mqtt-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make mqtt-linux; fi
	docker build -t $(IMAGE_PREFIX)mqtt-gateway:$(IMAGE_TAG) -f ./gateways/server/mqtt/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)mqtt-gateway:$(IMAGE_TAG) ; fi


# Custom gateways
storage-grid:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/storagegrid-gateway ./gateways/server/storagegrid/cmd

storage-grid-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make storage-grid

storage-grid-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make storage-grid-linux; fi
	docker build -t $(IMAGE_PREFIX)storage-grid-gateway:$(IMAGE_TAG) -f ./gateways/server/storagegrid/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)storage-grid-gateway:$(IMAGE_TAG) ; fi

gitlab:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gitlab-gateway ./gateways/server/gitlab/cmd

gitlab-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make gitlab

gitlab-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make gitlab-linux; fi
	docker build -t $(IMAGE_PREFIX)gitlab-gateway:$(IMAGE_TAG) -f ./gateways/server/gitlab/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gitlab-gateway:$(IMAGE_TAG) ; fi

github:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/github-gateway ./gateways/server/github/cmd

github-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make github

github-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make github-linux; fi
	docker build -t $(IMAGE_PREFIX)github-gateway:$(IMAGE_TAG) -f ./gateways/server/github/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)github-gateway:$(IMAGE_TAG) ; fi

sns:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/aws-sns-gateway ./gateways/server/aws-sns/cmd

sns-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sns

sns-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make sns-linux; fi
	docker build -t $(IMAGE_PREFIX)aws-sns-gateway:$(IMAGE_TAG) -f ./gateways/server/aws-sns/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)aws-sns-gateway:$(IMAGE_TAG) ; fi

pubsub:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/gcp-pubsub-gateway ./gateways/server/gcp-pubsub/cmd

pubsub-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make pubsub

pubsub-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make pubsub-linux; fi
	docker build -t $(IMAGE_PREFIX)gcp-pubsub-gateway:$(IMAGE_TAG) -f ./gateways/server/gcp-pubsub/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)gcp-pubsub-gateway:$(IMAGE_TAG) ; fi

hdfs:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/hdfs-gateway ./gateways/server/hdfs/cmd

hdfs-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make hdfs

hdfs-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make hdfs-linux; fi
	docker build -t $(IMAGE_PREFIX)hdfs-gateway:$(IMAGE_TAG) -f ./gateways/server/hdfs/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)hdfs-gateway:$(IMAGE_TAG) ; fi

sqs:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/aws-sqs-gateway ./gateways/server/aws-sqs/cmd

sqs-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make sqs

sqs-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make sqs-linux; fi
	docker build -t $(IMAGE_PREFIX)aws-sqs-gateway:$(IMAGE_TAG) -f ./gateways/server/aws-sqs/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)aws-sqs-gateway:$(IMAGE_TAG) ; fi

slack:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/slack-gateway ./gateways/server/slack/cmd

slack-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make slack

slack-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make slack-linux; fi
	docker build -t $(IMAGE_PREFIX)slack-gateway:$(IMAGE_TAG) -f ./gateways/server/slack/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)slack-gateway:$(IMAGE_TAG) ; fi

nsq:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/nsq-gateway ./gateways/server/nsq/cmd

nsq-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make nsq

nsq-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make nsq-linux; fi
	docker build -t $(IMAGE_PREFIX)nsq-gateway:$(IMAGE_TAG) -f ./gateways/server/nsq/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)nsq-gateway:$(IMAGE_TAG) ; fi

redis:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/redis-gateway ./gateways/server/redis/cmd

redis-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make redis

redis-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make redis-linux; fi
	docker build -t $(IMAGE_PREFIX)redis-gateway:$(IMAGE_TAG) -f ./gateways/server/redis/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)redis-gateway:$(IMAGE_TAG) ; fi

emitter:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/emitter-gateway ./gateways/server/emitter/cmd

emitter-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make emitter

emitter-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make emitter-linux; fi
	docker build -t $(IMAGE_PREFIX)emitter-gateway:$(IMAGE_TAG) -f ./gateways/server/emitter/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)emitter-gateway:$(IMAGE_TAG) ; fi

stripe:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/stripe-gateway ./gateways/server/stripe/cmd

stripe-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make stripe

stripe-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make stripe-linux; fi
	docker build -t $(IMAGE_PREFIX)stripe-gateway:$(IMAGE_TAG) -f ./gateways/server/stripe/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)stripe-gateway:$(IMAGE_TAG) ; fi

azure-events-hub:
	go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/azure-events-hub-gateway ./gateways/server/azure-events-hub/cmd

azure-events-hub-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make azure-events-hub

azure-events-hub-image:
	@if [ "$(BUILD_BINARY)" = "true" ]; then make azure-events-hub-linux; fi
	docker build -t $(IMAGE_PREFIX)azure-events-hub-gateway:$(IMAGE_TAG) -f ./gateways/server/azure-events-hub/Dockerfile .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_PREFIX)azure-events-hub-gateway:$(IMAGE_TAG) ; fi

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
build-e2e-images: sensor-controller-image gateway-controller-image gateway-client-image webhook-image

.PHONY: lint
lint:
	golangci-lint run

