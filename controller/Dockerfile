FROM alpine:3.7

COPY dist/sensor-controller /

# add the stream plugin binaries
COPY dist/plugins/ /plugins/

ENV STREAM_PLUGIN_DIR=plugins
RUN chmod -R 777 /plugins

CMD [ "/sensor-controller" ]
