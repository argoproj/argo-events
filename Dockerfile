####################################################################################################
# base
####################################################################################################
FROM alpine:3.12.3 as base
RUN apk update && apk upgrade && \
    apk add ca-certificates && \
    apk --no-cache add tzdata

ENV ARGO_VERSION=v3.0.2

RUN wget -q https://github.com/argoproj/argo/releases/download/${ARGO_VERSION}/argo-linux-amd64.gz
RUN gunzip argo-linux-amd64.gz
RUN chmod +x argo-linux-amd64
RUN mv ./argo-linux-amd64 /usr/local/bin/argo
RUN argo version

####################################################################################################
# argo-events
####################################################################################################
FROM scratch as argo-events
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=base /usr/local/bin/argo /usr/local/bin/argo
COPY dist/argo-events /bin/argo-events
ENTRYPOINT [ "/bin/argo-events" ]

