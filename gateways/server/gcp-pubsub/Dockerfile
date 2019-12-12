FROM centos:7
RUN yum -y install ca-certificates
COPY dist/gcp-pubsub-gateway /bin/
ENTRYPOINT [ "/bin/gcp-pubsub-gateway" ]
