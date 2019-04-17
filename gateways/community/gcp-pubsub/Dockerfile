FROM alpine:3.9 as alpine
RUN apk add -U --no-cache ca-certificates

FROM scratch
COPY dist/gcp-pubsub-gateway /bin/
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT [ "/bin/gcp-pubsub-gateway" ]
