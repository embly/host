FROM golang:alpine

WORKDIR /go/src/hello

COPY . .

RUN go build

FROM alpine

COPY --from=0 /go/src/hello/hello /bin/hello

CMD ["/bin/hello"]
