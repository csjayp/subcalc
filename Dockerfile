FROM ubuntu

ENV DEBIAN_FRONTEND=noninteractive
ENV GO_VERSION=1.24.0
ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

ADD . .
RUN apt-get -y update 
RUN apt-get -y install wget make gcc clang
RUN apt-get update && \
    apt-get install -y curl git build-essential ca-certificates && \
    rm -rf /var/lib/apt/lists/*
# Download and install Go
RUN curl -LO https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz

RUN wget https://github.com/fastly/cli/releases/download/v11.2.0/fastly_11.2.0_linux_amd64.deb
RUN dpkg -i fastly_11.2.0_linux_amd64.deb

ENTRYPOINT [ "/ci/build.sh" ]
