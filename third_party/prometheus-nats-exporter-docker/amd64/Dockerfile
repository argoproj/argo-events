# Golang binary building stage
FROM golang:1.20.2
WORKDIR $GOPATH/src/github.com/nats-io/prometheus-nats-exporter
RUN git clone --branch v0.10.1 https://github.com/nats-io/prometheus-nats-exporter.git .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a -tags netgo -installsuffix netgo -ldflags "-s -w"

# Final docker image building stage
FROM scratch
COPY --from=0 /go/src/github.com/nats-io/prometheus-nats-exporter/prometheus-nats-exporter /prometheus-nats-exporter
EXPOSE 7777
ENTRYPOINT ["/prometheus-nats-exporter"]
CMD ["--help"]
