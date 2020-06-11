FROM golang:alpine

RUN apk update && apk add go

WORKDIR /app
COPY main.go go.mod go.sum /app/

RUN mkdir /config
VOLUME /config

# Build binary
RUN go build -o firefly_gone_plaid

ENTRYPOINT ["/app/firefly_gone_plaid"]
