FROM centos:8
RUN yum -y update && yum -y install ca-certificates openssh openssh-server openssh-clients openssl-libs curl git

# Argo Workflow CLI
COPY assets/argo-linux-amd64 /usr/local/bin/argo

RUN argo version

COPY dist/sensor /bin/
ENTRYPOINT [ "/bin/sensor" ]
