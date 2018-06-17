FROM golang:1.7

RUN mkdir -p /go/src/github.com/bfirsh/whalebrew
WORKDIR /go/src/github.com/bfirsh/whalebrew

COPY . /go/src/github.com/bfirsh/whalebrew
RUN go-wrapper download -t ./... && \
  go-wrapper install
CMD ["go-wrapper", "run"]
