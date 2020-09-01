####################################################################################################
# Builder image
# Initial stage which pulls prepares build dependencies and CLI tooling we need for our final image
# Also used as the image in CI jobs so needs all dependencies
####################################################################################################
FROM golang:1.13.4 as builder

RUN apt-get update && apt-get --no-install-recommends install -y \
    git \
    make \
    apt-utils \
    apt-transport-https \
    ca-certificates \
    wget \
    gcc \
    zip && \
    apt-get clean \
    && rm -rf \
        /var/lib/apt/lists/* \
        /tmp/* \
        /var/tmp/* \
        /usr/share/man \
        /usr/share/doc \
        /usr/share/doc-base

WORKDIR /tmp

####################################################################################################
# Argo Build stage which performs the actual build of Argo binaries
####################################################################################################
FROM builder as build

ARG IMAGE_OS=linux

# Perform the build
WORKDIR /go/src/github.com/argoproj/argo-events
COPY . .
# check we can use Git
RUN git rev-parse HEAD

# Make sure there are not prebuilt binaries
RUN rm dist/*

# eventbus-controller
RUN . hack/image_arch.sh && make dist/eventbus-controller-linux-${IMAGE_ARCH}

# eventsource-controller
RUN . hack/image_arch.sh && make dist/eventsource-controller-linux-${IMAGE_ARCH}

# sensor-controller
RUN . hack/image_arch.sh && make dist/sensor-controller-linux-${IMAGE_ARCH}

# eventsource
RUN . hack/image_arch.sh && make dist/eventsource-linux-${IMAGE_ARCH}

# sensor
RUN . hack/image_arch.sh && make dist/sensor-linux-${IMAGE_ARCH}


####################################################################################################
# eventbus-controller
####################################################################################################
FROM scratch as eventbus-controller
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/src/github.com/argoproj/argo-events/dist/eventbus-controller /bin/eventbus-controller
ENTRYPOINT [ "/bin/eventbus-controller" ]

####################################################################################################
# eventsource-controller
####################################################################################################
FROM scratch as eventsource-controller
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/src/github.com/argoproj/argo-events/dist/eventsource-controller /bin/eventsource-controller
ENTRYPOINT [ "/bin/eventsource-controller" ]

####################################################################################################
# sensor-controller
####################################################################################################
FROM scratch as sensor-controller
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/src/github.com/argoproj/argo-events/dist/sensor-controller /bin/sensor-controller
ENTRYPOINT [ "/bin/sensor-controller" ]

####################################################################################################
# eventsource
####################################################################################################
FROM scratch as eventsource
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/src/github.com/argoproj/argo-events/dist/eventsource /bin/eventsource
ENTRYPOINT [ "/bin/eventsource" ]

####################################################################################################
# sensor
####################################################################################################
FROM alpine as sensor
RUN apk update && apk upgrade && \
    apk add --no-cache git

COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Argo Workflow CLI
COPY assets/argo-linux-amd64 /usr/local/bin/argo
RUN chmod +x /usr/local/bin/argo
RUN argo version || true

COPY --from=build /go/src/github.com/argoproj/argo-events/dist/sensor /bin/sensor

ENTRYPOINT [ "/bin/sensor" ]

