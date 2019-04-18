ARG GO_VERSION=1.12
ARG DOCKER_VERSION=18.06

FROM docker:${DOCKER_VERSION} as docker-cli
FROM golang:${GO_VERSION}-alpine as go
FROM go as build

WORKDIR /src

# build-base is required for tests
RUN apk add --no-cache \
    build-base \
    git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go test -v ./...
RUN go install .

FROM alpine
ENTRYPOINT ["whalebrew"]
LABEL io.whalebrew.config.volumes '["/var/run/docker.sock:/var/run/docker.sock", "${WHALEBREW_INSTALL_PATH}:/usr/local/bin"]'
LABEL io.whalebrew.config.keep_container_user 'true'

COPY --from=docker-cli /usr/local/bin/docker /bin/docker
COPY --from=build /go/bin/whalebrew /bin/whalebrew
