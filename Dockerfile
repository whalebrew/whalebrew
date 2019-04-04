FROM golang:1.11

RUN mkdir -p /go/src/github.com/whalebrew/whalebrew
WORKDIR /go/src/github.com/whalebrew/whalebrew

COPY . /go/src/github.com/whalebrew/whalebrew
RUN go get -t -v ./...
RUN go install .
CMD ["whalebrew"]
