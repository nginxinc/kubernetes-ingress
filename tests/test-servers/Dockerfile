# syntax=docker/dockerfile:1.3
FROM golang:1.17-alpine as builder

RUN mkdir /app

ARG type

COPY $type/main.go /app

WORKDIR /app

RUN GO111MODULE=off CGO_ENABLED=0 GOOS=linux go build -o main

FROM scratch

COPY --from=builder /app/main /app/

ENTRYPOINT ["/app/main"]
