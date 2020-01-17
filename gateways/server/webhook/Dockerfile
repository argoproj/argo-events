FROM centos:7
RUN yum -y install ca-certificates
COPY dist/webhook-gateway /bin/
ENTRYPOINT [ "/bin/webhook-gateway" ]