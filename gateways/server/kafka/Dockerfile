FROM centos:7
RUN yum -y install ca-certificates
COPY dist/kafka-gateway /bin/
ENTRYPOINT [ "/bin/kafka-gateway" ]