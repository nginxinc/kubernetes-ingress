FROM golang:1.16-alpine as builder

RUN mkdir /app

COPY main.go /app

WORKDIR /app

RUN GO111MODULE=off CGO_ENABLED=0 GOOS=linux go build -o main

FROM scratch

COPY --from=builder /app/main /app/

EXPOSE 3333

ENTRYPOINT ["/app/main"]
