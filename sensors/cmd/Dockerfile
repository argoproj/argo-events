FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Argo Workflow CLI
COPY assets/argo-linux-amd64 /bin/argo
COPY dist/sensor /bin/
ENTRYPOINT [ "/bin/sensor" ]
