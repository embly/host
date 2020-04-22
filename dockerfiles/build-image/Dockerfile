FROM golang:1.14-alpine

RUN apk add build-base iptables ip6tables
WORKDIR /host
COPY go.mod go.sum ./
RUN go mod download
