FROM golang:1.22-alpine

WORKDIR /app

RUN apk update && \
    apk add --no-cache \
      curl==8.5.0-r0 \
      bash==5.2.21-r0 \
      ca-certificates==20240226-r0 && \
    update-ca-certificates

COPY . ./
RUN go mod download && \
  go build -o /app/scyllaridae && \
  go clean -cache -modcache

ENTRYPOINT ["/app/scyllaridae"]
