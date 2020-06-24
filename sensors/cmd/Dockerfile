FROM centos:7

ARG ARGO_VERSION=v2.8.2

RUN yum -y install ca-certificates openssh openssh-server openssh-clients openssl-libs curl git

# OpenFass CLI
# TODO: donwload faas instead of copying it from the svc, same as with argo-cli
COPY assets/faas-cli /usr/local/bin/faas

# Argo Workflow CLI
RUN curl -sSL -o /usr/local/bin/argo \
    https://github.com/argoproj/argo/releases/download/${ARGO_VERSION}/argo-linux-amd64 \
 && chmod +x /usr/local/bin/argo

# added `|| true` to ignore "invalid configuration: no configuration has been provided"
# TODO: argo version shouldn't rely on a cluster connection to execute
RUN argo version || true
RUN faas version

RUN mkdir /.openfaas
RUN chmod 777 /.openfaas

RUN mkdir /bin/workflows
RUN chmod -R 777 /bin/workflows

COPY dist/sensor /bin/
ENTRYPOINT [ "/bin/sensor" ]
