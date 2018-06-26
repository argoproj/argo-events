FROM alpine:3.7

COPY dist/sensor-controller /

# add the stream plugin binary
COPY dist/stream-plugin /
ENV STREAM_PLUGIN=/stream-plugin

CMD [ "/sensor-controller" ]
