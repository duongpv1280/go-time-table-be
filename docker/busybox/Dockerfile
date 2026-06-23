FROM golang:1.24.0-bullseye

WORKDIR /go/src

ENV GO111MODULE=on \
    GOFLAGS='-buildvcs=false' \
    CGO_ENABLED=0 \
    GOOS=linux \
    SQL_MIGRATE_VERSION=v1.1.2 \
    STRICTGOIMPORTS_VERSION=v1.2.1 \
    OAPI_CODEGEN_VERSION=v2.3.0 \
    MOCKGEN_VERSION=v1.6.0 \
    YQ_VERSION=v4.23.1 \
    PNPM_VERSION=v7.9.5 \
    WIRE_VERSION=v0.6.0 \
    AIR_VERSION=v1.27.3 \
    DLV_VERSIOON=v1.25.0 \
    GOMOCKHANDLER_VERSION=v1.4.4 \
    GOTESTSUM_VERSION=v1.11.0 \
    GOMOCKERY_VERSION=v2.46.3 \
    GQLGENC_VERSION=v0.18.1 \
    OPENAPI_GENERATOR_VERSION=6.0.1 \
    MIGRATE_VERSION=v4.18.0

RUN set -eux && \
    apt-get update && \
    apt-get install -y curl lsb-release default-jdk poppler-utils

RUN curl -fsSL https://deb.nodesource.com/setup_18.x | bash - && apt-get install -y nodejs

RUN npm install -g @apidevtools/swagger-cli && \
    npm install -g @openapitools/openapi-generator-cli && \
    npx @openapitools/openapi-generator-cli version-manager set ${OPENAPI_GENERATOR_VERSION} && \
    mkdir -p /root/.openapi-generator && \
    cd /root/.openapi-generator && \
    curl -0 https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/${OPENAPI_GENERATOR_VERSION}/openapi-generator-cli-${OPENAPI_GENERATOR_VERSION},jar && \
    echo '{"$schema":"./node_modules/@openapitools/openapi-generator-cli/config/schema.json","spaces":2,"generator-cli":{"version":"'${OPENAPI_GENERATOR_VERSION}'"}}' > /root/.openapi-generator-cli.json

RUN npm i -g openapi-to-postmanv2

COPY go.mod go.sum ./
RUN go mod download

RUN go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@${MIGRATE_VERSION} && \
    go install github.com/rubenv/sql-migrate/...@${SQL_MIGRATE_VERSION} && \
    go install github.com/momotaro98/strictgoimports/cmd/strictgoimports@${STRICTGOIMPORTS_VERSION} && \
    go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION} && \
    go install github.com/mikefarah/yq/v4@${YQ_VERSION} && \
    go install github.com/golang/mock/mockgen@${MOCKGEN_VERSION} && \
    go install github.com/google/wire/cmd/wire@${WIRE_VERSION} && \
    go install github.com/cosmtrek/air@${AIR_VERSION} && \
    go install github.com/go-delve/delve/cmd/dlv@${DLV_VERSIOON} && \
    go install github.com/sanposhiho/gomockhandler@${GOMOCKHANDLER_VERSION} && \
    go install gotest.tools/gotestsum@${GOTESTSUM_VERSION} && \
    go install github.com/Yamashou/gqlgenc@${GQLGENC_VERSION} && \
    go install github.com/vektra/mockery/v2@${GOMOCKERY_VERSION}