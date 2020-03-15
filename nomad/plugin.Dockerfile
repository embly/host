FROM golang:buster

RUN apt-get update && apt-get install -y make git

WORKDIR /go/src/github.com/hashicorp/nomad
COPY . .
RUN make bootstrap
RUN cd drivers/docker/cmd && go build -o custom-docker
