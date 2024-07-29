FROM golang:1.22-alpine

WORKDIR /app

SHELL ["/bin/ash", "-o", "pipefail", "-c"]

RUN apk update && \
    apk add --no-cache \
      curl==8.9.0-r0 \
      bash==5.2.26-r0 \
      ca-certificates==20240705-r0 \
      openssl==3.3.1-r3 && \
    openssl s_client -connect helloworld.letsencrypt.org:443 -showcerts </dev/null 2>/dev/null | sed -e '/-----BEGIN/,/-----END/!d' | tee "/usr/local/share/ca-certificates/letsencrypt.crt" >/dev/null && \
    update-ca-certificates

COPY . ./
RUN go mod download && \
  go build -o /app/scyllaridae && \
  go clean -cache -modcache

ENTRYPOINT ["/bin/bash"]
CMD ["/app/docker-entrypoint.sh"]
