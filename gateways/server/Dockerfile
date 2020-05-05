FROM centos:7
RUN yum -y install ca-certificates
COPY dist/gateway-server /bin/
ENTRYPOINT [ "/bin/gateway-server" ]