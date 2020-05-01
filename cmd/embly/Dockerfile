# syntax = docker/dockerfile:experimental
FROM golang:1.14-alpine

WORKDIR /opt
RUN apk add build-base
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY cmd ./cmd
COPY pkg ./pkg

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd cmd/cli && go build

FROM alpine:3.11
RUN apk add bash curl iptables
COPY --from=0 /opt/cmd/cli/cli /bin/cli
CMD ["/bin/cli","agent"]
