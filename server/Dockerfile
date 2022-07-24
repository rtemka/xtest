# syntax=docker/dockerfile:1

FROM golang:1.18-buster As build
LABEL version="1.0.0" maintainer="Artem Rybakov<rybakov333@gmail.com>" 

WORKDIR /go/src/github.com/rtemka/xtest

COPY . .

RUN go mod tidy 

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o ./cmd/server ./cmd/main.go

FROM alpine:3.16.1

WORKDIR /app

COPY --from=build go/src/github.com/rtemka/xtest/cmd/server .

EXPOSE 8080 8090

ENTRYPOINT [ "./server"]