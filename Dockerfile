FROM golang:1.7

RUN mkdir -p /go/src/github.com/whalebrew/whalebrew
WORKDIR /go/src/github.com/whalebrew/whalebrew

COPY . /go/src/github.com/whalebrew/whalebrew
RUN go-wrapper download -t ./...
RUN go-wrapper install
CMD ["go-wrapper", "run"]
