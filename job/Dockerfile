FROM alpine:3.7

COPY dist/sensor-executor /

COPY dist/signal-plugin /
ENV SIGNAL_PLUGIN=/signal-plugin

CMD [ "/sensor-executor" ]
