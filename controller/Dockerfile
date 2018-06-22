FROM alpine:3.7

COPY dist/sensor-controller /

COPY dist/signal-plugin /
ENV SIGNAL_PLUGIN=/signal-plugin

CMD [ "/sensor-controller" ]
