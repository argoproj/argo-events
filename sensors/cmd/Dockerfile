FROM centos:7
RUN yum -y install openssh openssh-server openssh-clients openssl-libs
# Download the binary
RUN curl -sLO https://github.com/argoproj/argo/releases/download/v2.4.3/argo-darwin-amd64
# Make binary executable
RUN chmod +x argo-darwin-amd64
# Move binary to path
RUN mv ./argo-darwin-amd64 /usr/local/bin/argo
RUN argo version
COPY dist/sensor /bin/
ENTRYPOINT [ "/bin/sensor" ]