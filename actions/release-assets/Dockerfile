FROM golang:1.18-alpine as build
RUN apk add --no-cache git build-base
RUN mkdir /src
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY main.go ./
RUN go build -o /bin/release .
COPY main_test.go ./
COPY resources ./resources
RUN go test -v ./...

FROM alpine
COPY --from=build /bin/release /bin/release
ENTRYPOINT ["/bin/release"]