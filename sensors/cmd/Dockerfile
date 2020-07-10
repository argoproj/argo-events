FROM centos:8
RUN yum -y update && yum -y install ca-certificates openssh openssh-server openssh-clients openssl-libs curl git

# OpenFass CLI
COPY assets/faas-cli /usr/local/bin/faas

# Argo Workflow CLI
COPY assets/argo-linux-amd64 /usr/local/bin/argo

RUN argo version
RUN faas version

RUN mkdir /.openfaas
RUN chmod 777 /.openfaas

RUN mkdir /bin/workflows
RUN chmod -R 777 /bin/workflows

COPY dist/sensor /bin/
ENTRYPOINT [ "/bin/sensor" ]
