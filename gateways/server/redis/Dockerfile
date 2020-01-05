FROM centos:7
RUN yum -y install ca-certificates
COPY dist/redis-gateway /bin/
ENTRYPOINT [ "/bin/redis-gateway" ]