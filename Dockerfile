FROM golang:1.24.0-alpine as build

WORKDIR /go/src

run set -eux && \
    apk update

ARG ENVIRONMENT

ENV GO111MODULE=on
COPY go.mod go.sum ./
RUN go mod download
COPY . .

CMD ["api", "-c", ".air/.air.toml"]