FROM centos:7
RUN yum -y install ca-certificates
COPY dist/azure-events-hub-gateway /bin/
ENTRYPOINT ["/bin/azure-events-hub-gateway"]
