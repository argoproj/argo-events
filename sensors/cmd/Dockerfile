FROM centos:7
RUN yum -y install openssh openssh-server openssh-clients openssl-libs
COPY dist/sensor /bin/
ENTRYPOINT [ "/bin/sensor" ]