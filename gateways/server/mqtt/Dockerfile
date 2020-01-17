FROM centos:7
RUN yum -y install ca-certificates
COPY dist/mqtt-gateway /bin/
ENTRYPOINT [ "/bin/mqtt-gateway" ]