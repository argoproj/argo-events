FROM centos:7
RUN yum -y install ca-certificates
COPY dist/emitter-gateway /bin/
ENTRYPOINT [ "/bin/emitter-gateway" ]