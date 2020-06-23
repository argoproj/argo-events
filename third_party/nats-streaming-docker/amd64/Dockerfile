FROM scratch

COPY nats-streaming-server /nats-streaming-server

# Expose client and management ports
EXPOSE 4222 8222

# Run with default memory based store 
ENTRYPOINT ["/nats-streaming-server"]
CMD ["-m", "8222"]
