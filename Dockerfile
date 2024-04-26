FROM golang:1.21-alpine

WORKDIR /app

RUN apk update && \
    apk add openssl && \
  openssl s_client -connect helloworld.letsencrypt.org:443 -showcerts </dev/null 2>/dev/null | sed -e '/-----BEGIN/,/-----END/!d' | tee "/usr/local/share/ca-certificates/ca.crt" >/dev/null && \
  update-ca-certificates

COPY . ./
RUN go mod download && \
  go build -o /app/scyllaridae && \
  go clean -cache -modcache

ENTRYPOINT ["/app/scyllaridae"]
