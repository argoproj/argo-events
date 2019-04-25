FROM centos:7
RUN yum -y install ca-certificates
COPY dist/aws-sns-gateway /bin/
ENTRYPOINT [ "/bin/aws-sns-gateway" ]
