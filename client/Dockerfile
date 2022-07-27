# syntax=docker/dockerfile:1

FROM golang:1.18-alpine3.16 As build
LABEL version="1.0.0" maintainer="Artem Rybakov<rybakov333@gmail.com>" 

WORKDIR /go/src/github.com/rtemka/xtest

COPY . .

RUN go mod tidy 

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o ./client ./client.go

FROM scratch

WORKDIR /app

COPY --from=build go/src/github.com/rtemka/xtest/client .

ENTRYPOINT [ "./client"]