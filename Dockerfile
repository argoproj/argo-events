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

USER 9731
ENTRYPOINT [ "/bin/eventbus-controller" ]

####################################################################################################
# eventsource-controller
####################################################################################################
FROM scratch as eventsource-controller
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/eventsource-controller /bin/eventsource-controller

USER 9731
ENTRYPOINT [ "/bin/eventsource-controller" ]

####################################################################################################
# sensor-controller
####################################################################################################
FROM scratch as sensor-controller
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/sensor-controller /bin/sensor-controller

USER 9731
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

COPY dist/sensor /bin/sensor

ENTRYPOINT [ "/bin/sensor" ]
