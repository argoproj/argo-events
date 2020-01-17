FROM centos:7
RUN yum -y install ca-certificates
COPY dist/storagegrid-gateway /bin/
ENTRYPOINT [ "/bin/storagegrid-gateway" ]