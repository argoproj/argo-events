FROM alpine:3.7

COPY dist/sensor-controller /

COPY dist/signal-plugin /
ENV SIGNAL_PLUGIN=/signal-plugin

# add more plugin binaries here
COPY dist/calendar-plugin /
ENV CAL_PLUGIN=/calendar-plugin

CMD [ "/sensor-controller" ]
