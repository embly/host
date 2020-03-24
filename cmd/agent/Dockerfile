FROM golang:1.14-alpine

WORKDIR /opt

COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY pkg ./pkg

RUN cd cmd/agent && go build

FROM alpine

RUN apk add bash curl iptables
COPY --from=0 /opt/cmd/agent/agent /bin/agent

CMD ["/bin/agent"]
