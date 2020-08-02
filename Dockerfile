####################################################################################################
# Builder image
####################################################################################################
FROM golang:1.13.4 as builder

ARG IMAGE_OS=linux
ARG IMAGE_ARCH=amd64

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

# Install docker
ENV DOCKER_CHANNEL stable
ENV DOCKER_VERSION 18.09.1

RUN if [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "amd64" ]; then \
    	wget -O docker.tgz https://download.docker.com/linux/static/${DOCKER_CHANNEL}/x86_64/docker-${DOCKER_VERSION}.tgz; \
    elif [ "${IMAGE_OS}" = "linux" -a "${IMAGE_ARCH}" = "arm64" ]; then \
	wget -O docker.tgz https://download.docker.com/linux/static/${DOCKER_CHANNEL}/aarch64/docker-${DOCKER_VERSION}.tgz; \
    fi && \
    tar --extract --file docker.tgz --strip-components 1 --directory /usr/local/bin/ && \
    rm docker.tgz

####################################################################################################
# Build base
####################################################################################################
FROM builder as argo-events-build-base

# Perform the build
WORKDIR /argo-events
COPY . .
# check we can use Git
RUN git rev-parse HEAD

####################################################################################################
# Build eventbus-controller binary
####################################################################################################
FROM argo-events-build-base as eventbus-controller-build

# eventbus-controller image
RUN make eventbus-controller-linux

####################################################################################################
# Build eventsource-controller binary
####################################################################################################
FROM argo-events-build-base as eventsource-controller-build

# eventsource-controller image
RUN make eventsource-controller-linux

####################################################################################################
# Build sensor-controller binary
####################################################################################################
FROM argo-events-build-base as sensor-controller-build

# sensor-controller image
RUN make sensor-controller-linux

####################################################################################################
# Build eventsource binary
####################################################################################################
FROM argo-events-build-base as eventsource-build

# eventsource image
RUN make eventsource-linux

####################################################################################################
# Build sensor binary
####################################################################################################
FROM argo-events-build-base as sensor-build

# sensor image
RUN make sensor-linux

####################################################################################################
# certs
####################################################################################################
FROM alpine:latest as certs
RUN apk --update add ca-certificates

####################################################################################################
# eventbus-controller
####################################################################################################
FROM scratch as eventbus-controller
# Add timezone data
COPY --from=eventbus-controller-build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=eventbus-controller-build /argo-events/dist/eventbus-controller /bin/eventbus-controller
ENTRYPOINT [ "/bin/eventbus-controller" ]

####################################################################################################
# eventsource-controller
####################################################################################################
FROM scratch as eventsource-controller
# Add timezone data
COPY --from=eventsource-controller-build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=eventsource-controller-build /argo-events/dist/eventsource-controller /bin/eventsource-controller
ENTRYPOINT [ "/bin/eventsource-controller" ]

####################################################################################################
# sensor-controller
####################################################################################################
FROM scratch as sensor-controller
# Add timezone data
COPY --from=sensor-controller-build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=sensor-controller-build /argo-events/dist/sensor-controller /bin/sensor-controller
ENTRYPOINT [ "/bin/sensor-controller" ]

####################################################################################################
# eventsource
####################################################################################################
FROM scratch as eventsource
# Add timezone data
COPY --from=eventsource-build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=eventsource-build /argo-events/dist/eventsource /bin/eventsource
ENTRYPOINT [ "/bin/eventsource" ]

####################################################################################################
# sensor
####################################################################################################
FROM centos:8 as sensor
RUN yum -y update && yum -y install ca-certificates openssh openssh-server openssh-clients openssl-libs curl git

# Argo Workflow CLI
COPY assets/argo-linux-amd64 /usr/local/bin/argo
RUN argo version || true

COPY --from=sensor-build /argo-events/dist/sensor /bin/sensor

ENTRYPOINT [ "/bin/sensor" ]

