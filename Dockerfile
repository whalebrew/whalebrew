FROM golang:1.12

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GO111MODULE=on go install .
CMD ["whalebrew"]
