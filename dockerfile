# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.18.2 AS builder

WORKDIR /src

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN cd ./alex && go build -o /app

##
## Deploy
##
FROM alpine

WORKDIR /

COPY --from=builder /app /app

EXPOSE 8080

ENTRYPOINT ["/app"]
