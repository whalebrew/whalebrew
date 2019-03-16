FROM golang:1.11-alpine AS src

RUN mkdir -p /src/
WORKDIR /src


RUN apk add --no-cache git build-base

# cache vendor to re-download vendor dependencies only when go.* changes
COPY go.* /src/

RUN go mod download

COPY . /src

FROM src as build
RUN go build -o /go/bin/whalebrew

FROM alpine
COPY --from=docker:18.09 /usr/local/bin/docker /usr/local/bin/docker
COPY --from=build /go/bin/whalebrew /usr/local/bin/whalebrew
ENTRYPOINT ["/usr/local/bin/whalebrew"]