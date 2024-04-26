FROM golang:1.21-alpine

WORKDIR /app

RUN apk update && \
    apk add bash ca-certificates && \
    update-ca-certificates

COPY . ./
RUN go mod download && \
  go build -o /app/scyllaridae && \
  go clean -cache -modcache

ENTRYPOINT ["/app/scyllaridae"]
