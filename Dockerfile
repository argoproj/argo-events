####################################################################################################
# base
####################################################################################################
FROM golang:alpine as base
RUN apk --update add ca-certificates
RUN apk --no-cache add tzdata

####################################################################################################
# eventbus-controller
####################################################################################################
FROM scratch as eventbus-controller
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/eventbus-controller /bin/eventbus-controller

ENTRYPOINT [ "/bin/eventbus-controller" ]

####################################################################################################
# eventsource-controller
####################################################################################################
FROM scratch as eventsource-controller
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/eventsource-controller /bin/eventsource-controller

ENTRYPOINT [ "/bin/eventsource-controller" ]

####################################################################################################
# sensor-controller
####################################################################################################
FROM scratch as sensor-controller
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/sensor-controller /bin/sensor-controller

ENTRYPOINT [ "/bin/sensor-controller" ]

####################################################################################################
# eventsource
####################################################################################################
FROM scratch as eventsource
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/eventsource /bin/eventsource
ENTRYPOINT [ "/bin/eventsource" ]

####################################################################################################
# sensor-base
####################################################################################################
FROM alpine:3.12.3 as sensor-base
RUN apk update && apk upgrade && \
    apk add --no-cache git

ENV ARGO_VERSION=v3.0.2

RUN wget -q https://github.com/argoproj/argo/releases/download/${ARGO_VERSION}/argo-linux-amd64.gz
RUN gunzip argo-linux-amd64.gz
RUN chmod +x argo-linux-amd64
RUN mv ./argo-linux-amd64 /usr/local/bin/argo
RUN argo version

####################################################################################################
# sensor
####################################################################################################
#FROM alpine:3.12.3 as sensor
#RUN apk update && apk upgrade && \
#    apk add --no-cache git
FROM scratch as sensor


COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=sensor-base /usr/local/bin/argo /usr/local/bin/argo
#RUN /usr/local/bin/argo version

COPY dist/sensor /bin/sensor

ENTRYPOINT [ "/bin/sensor" ]

####################################################################################################
# events-webhook
####################################################################################################
FROM scratch as events-webhook
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/events-webhook /bin/events-webhook
ENTRYPOINT [ "/bin/events-webhook" ]

