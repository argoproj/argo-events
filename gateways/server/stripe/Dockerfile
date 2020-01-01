FROM centos:7
RUN yum -y install ca-certificates
COPY dist/stripe-gateway /bin/
ENTRYPOINT [ "/bin/stripe-gateway" ]