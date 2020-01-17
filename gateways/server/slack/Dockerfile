FROM centos:7
RUN yum -y install ca-certificates
COPY dist/slack-gateway /bin/
ENTRYPOINT [ "/bin/slack-gateway" ]