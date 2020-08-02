####################################################################################################
# certs
####################################################################################################
FROM alpine:latest as certs
RUN apk --update add ca-certificates

####################################################################################################
# build base
####################################################################################################
FROM golang:alpine as build-base
RUN apk --no-cache add tzdata

####################################################################################################
# eventbus-controller
####################################################################################################
FROM scratch as eventbus-controller
# Add timezone data
COPY --from=build-base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/eventbus-controller /bin/eventbus-controller
ENTRYPOINT [ "/bin/eventbus-controller" ]

####################################################################################################
# eventsource-controller
####################################################################################################
FROM scratch as eventsource-controller
# Add timezone data
COPY --from=build-base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/eventsource-controller /bin/eventsource-controller
ENTRYPOINT [ "/bin/eventsource-controller" ]

####################################################################################################
# sensor-controller
####################################################################################################
FROM scratch as sensor-controller
# Add timezone data
COPY --from=build-base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/sensor-controller /bin/sensor-controller
ENTRYPOINT [ "/bin/sensor-controller" ]

####################################################################################################
# eventsource
####################################################################################################
FROM scratch as eventsource
# Add timezone data
COPY --from=build-base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/eventsource /bin/eventsource
ENTRYPOINT [ "/bin/eventsource" ]

####################################################################################################
# sensor
####################################################################################################
FROM centos:8 as sensor
RUN yum -y update && yum -y install ca-certificates openssh openssh-server openssh-clients openssl-libs curl git

# Argo Workflow CLI
COPY assets/argo-linux-amd64 /usr/local/bin/argo
RUN argo version || true

COPY dist/sensor /bin/sensor

ENTRYPOINT [ "/bin/sensor" ]

