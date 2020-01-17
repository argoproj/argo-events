FROM centos:7
RUN yum -y install ca-certificates
COPY dist/amqp-gateway /bin/
ENTRYPOINT [ "/bin/amqp-gateway" ]