FROM centos:7
RUN yum -y install ca-certificates
COPY dist/minio-gateway /bin/
ENTRYPOINT [ "/bin/minio-gateway" ]