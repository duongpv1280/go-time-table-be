FROM golang:1.24.0-alpine

WORKDIR /go/src

ENV GO111MODULE=on \
    GOFLAGS='-buildvcs=false' \
    CGO_ENABLED=0 \
    GOOS=linux \
    TEST_AWS_ID=root \
    TEST_AWS_SECRET=password \
    TEST_AWS_ENDPOINT=http://127.0.0.1:9000 \
    TEST_AWS_REGION=ap-northeast-1 \
    TEST_AWS_BUCKET_NAME=local-go-time-table \
    OAPI_CODEGEN_VERSION=v2.3.0 \
    MOCKGEN_VERSION=v1.6.0 \
    WIRE_VERSION=v0.6.0 \
    GOMOCKHANDLER_VERSION=v1.4.4 \
    GOTESTSUM_VERSION=v1.11.0 \

RUN set -eux && \
    apk update && \
    apk add --no-cache curl npm openjdk17 poppler-utils

RUN npm install -g @apidevtools/swagger-cli && \
    npm install -g @openapitools/openapi-generator-cli &&
RUN npm i -g openapi-to-postmanv2

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION} && \
    go install github.com/golang/mock/mockgen@${MOCKGEN_VERSION} && \
    go install github.com/google/wire/cmd/wire@${WIRE_VERSION} && \
    go install github.com/sanposhiho/gomockhandler@${GOMOCKHANDLER_VERSION} && \
    go install gotest.tools/gotestsum@${GOTESTSUM_VERSION}