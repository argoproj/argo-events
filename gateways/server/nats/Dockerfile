FROM centos:7
RUN yum -y install ca-certificates
COPY dist/nats-gateway /bin/
ENTRYPOINT [ "/bin/nats-gateway" ]