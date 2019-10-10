FROM centos:7
RUN yum -y install ca-certificates
COPY dist/gitlab-gateway /bin/
ENTRYPOINT [ "/bin/gitlab-gateway" ]