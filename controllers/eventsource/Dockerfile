FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY dist/eventsource-controller /bin/
ENTRYPOINT [ "/bin/eventsource-controller" ]
