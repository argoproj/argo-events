# Need to use an alpine image because go-plugin requires the use of tmp file directory
FROM alpine:3.7

COPY dist/sensor-controller /

COPY dist/signal-plugin /
ENV SIGNAL_PLUGIN=/signal-plugin

CMD [ "/sensor-controller" ]
