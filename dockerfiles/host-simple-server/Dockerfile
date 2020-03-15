FROM golang:latest

WORKDIR /go/src/hello

COPY . .

RUN go build && strip hello

FROM debian:buster

RUN apt-get update
RUN apt-get install -y --no-install-recommends bash curl iptables
COPY --from=0 /go/src/hello/hello /bin/hello

CMD ["/bin/hello"]
