FROM centos:7
RUN yum -y install ca-certificates
COPY dist/nsq-gateway /bin/
ENTRYPOINT [ "/bin/nsq-gateway" ]
