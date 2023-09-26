FROM golang:1.20-alpine as build
RUN apk add --no-cache git build-base
RUN mkdir /src
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY main.go ./
COPY pkg ./pkg/
RUN go build -o /bin/release .

FROM alpine
RUN apk add --no-cache git
COPY --from=build /bin/release /bin/release
ENTRYPOINT ["/bin/release"]