FROM golang:latest

RUN mkdir -p $GOPATH/src/golang.org/x/ \
    && cd $GOPATH/src/golang.org/x/ \
    && git clone https://github.com/golang/net.git \
    && go get github.com/go-sql-driver/mysql \
    && go get github.com/docker/docker/client \
    && go get github.com/docker/docker/api/types

ADD ["build.sh", "/"]

WORKDIR /