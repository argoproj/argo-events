FROM argoproj/argo-ci-builder:1.0

RUN apt-get update && apt-get install unzip
RUN mkdir /tmp/protobuf && cd /tmp/protobuf && curl -sOL https://github.com/google/protobuf/releases/download/v3.3.0/protoc-3.3.0-linux-x86_64.zip && unzip protoc-3.3.0-linux-x86_64.zip && cp bin/protoc /usr/local/bin/protoc && mv include /usr/local/
