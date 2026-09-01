ARG ARCH=$TARGETARCH
ARG ARGO_VERSION=v3.7.9

####################################################################################################
# common-builder
####################################################################################################
FROM alpine:3.23 AS builder

ARG ARCH
ARG ARGO_VERSION

RUN apk add --no-cache \
        ca-certificates \
        tzdata \
        wget
RUN wget -q https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/argo-linux-${ARCH}.gz \
    && gunzip -f argo-linux-${ARCH}.gz \
    && chmod +x argo-linux-${ARCH} \
    && mv argo-linux-${ARCH} /usr/local/bin/argo
COPY dist/argo-events-linux-${ARCH} /bin/argo-events
RUN chmod +x /bin/argo-events
####################################################################################################
# Common non-root builder
####################################################################################################
FROM builder AS builder-non-root

RUN addgroup -g 8737 argo-events \
 && adduser -D -u 8737 -G argo-events argo-events \
 && chown 8737:8737 /usr/local/bin/argo /bin/argo-events
####################################################################################################
# argo-events
####################################################################################################
FROM scratch AS argo-events

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/local/bin/argo /usr/local/bin/argo
COPY --from=builder /bin/argo-events /bin/argo-events

ENTRYPOINT [ "/bin/argo-events" ]

####################################################################################################
# argo-events-non-root
####################################################################################################
FROM builder-non-root AS argo-events-non-root

# Run as non-root user (UID 8737)
USER 8737
ENTRYPOINT [ "/bin/argo-events" ]
