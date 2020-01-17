FROM golang:1.12-stretch

RUN apt-get install -y ca-certificates

RUN apt-get update \
 && apt-get upgrade -y \
 && apt-get install -y git


RUN curl -kLo /usr/local/bin/dep \
             https://github.com/golang/dep/releases/download/v0.5.4/dep-linux-amd64 \
     && chmod +x /usr/local/bin/dep

COPY ../. .

RUN ls -la

