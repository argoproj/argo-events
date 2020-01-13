FROM centos:7
RUN yum -y install ca-certificates openssh openssh-server openssh-clients openssl-libs curl

# OpenFass CLI
COPY assets/faas-cli /usr/local/bin/faas

# Argo Workflow CLI
COPY assets/argo-linux-amd64 /usr/local/bin/argo

RUN argo version
RUN faas version

RUN mkdir /.openfaas
RUN chmod 777 /.openfaas

COPY dist/sensor /bin/
ENTRYPOINT [ "/bin/sensor" ]
