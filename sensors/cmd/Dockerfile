FROM centos:7
RUN yum -y install openssh openssh-server openssh-clients openssl-libs

# Argo Workflow CLI
# Download the binary
RUN curl -sLO https://github.com/argoproj/argo/releases/download/v2.4.3/argo-linux-amd64
# Make binary executable
RUN chmod +x argo-linux-amd64
# Move binary to path
RUN mv ./argo-linux-amd64 /usr/local/bin/argo
RUN argo version

# OpenFass CLI
RUN curl -sSL https://cli.openfaas.com > faas
# Make binary executable
RUN chmod +x faas
# Move binary to path
RUN mv ./faas /usr/local/bin/faas
RUN faas version

COPY dist/sensor /bin/
ENTRYPOINT [ "/bin/sensor" ]