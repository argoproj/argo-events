FROM centos:7
RUN yum -y install ca-certificates
COPY dist/hdfs-gateway /bin/
ENTRYPOINT [ "/bin/hdfs-gateway" ]
