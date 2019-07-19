FROM ubuntu
ADD . .
RUN apt-get -y update 
RUN apt-get -y install make gcc clang 
ENTRYPOINT [ "/ci/build.sh" ]
