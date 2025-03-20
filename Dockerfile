ARG ARCH=$TARGETARCH
####################################################################################################
# base
####################################################################################################
FROM alpine:3.16.2 as base
ARG ARCH
RUN apk update && apk upgrade && \
    apk add ca-certificates && \
    apk --no-cache add tzdata

ENV ARGO_VERSION=v3.6.5

RUN wget -q https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/argo-linux-${ARCH}.gz
RUN gunzip -f argo-linux-${ARCH}.gz
RUN chmod +x argo-linux-${ARCH}
RUN mv ./argo-linux-${ARCH} /usr/local/bin/argo
COPY dist/argo-events-linux-${ARCH} /bin/argo-events
RUN chmod +x /bin/argo-events

####################################################################################################
# argo-events
####################################################################################################
FROM scratch as argo-events
ARG ARCH
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=base /usr/local/bin/argo /usr/local/bin/argo
COPY --from=base /bin/argo-events /bin/argo-events
ENTRYPOINT [ "/bin/argo-events" ]
