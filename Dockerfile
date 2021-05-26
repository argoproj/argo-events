ARG ARCH=$TARGETARCH
####################################################################################################
# base
####################################################################################################
FROM alpine:3.12.3 as base
ARG ARCH
RUN apk update && apk upgrade && \
    apk add ca-certificates && \
    apk --no-cache add tzdata

ENV ARGO_VERSION=v3.0.2

RUN wget -q https://github.com/argoproj/argo/releases/download/${ARGO_VERSION}/argo-linux-${ARCH}.gz
RUN gunzip argo-linux-${ARCH}.gz
RUN chmod +x argo-linux-${ARCH}
RUN mv ./argo-linux-${ARCH} /usr/local/bin/argo

####################################################################################################
# argo-events
####################################################################################################
FROM scratch as argo-events
ARG ARCH
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=base /usr/local/bin/argo /usr/local/bin/argo
COPY dist/argo-events-linux-${ARCH} /bin/argo-events
ENTRYPOINT [ "/bin/argo-events" ]
