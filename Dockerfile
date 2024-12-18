FROM golang:1.23-alpine3.20@sha256:9a31ef0803e6afdf564edc8ba4b4e17caed22a0b1ecd2c55e3c8fdd8d8f68f98

SHELL ["/bin/ash", "-o", "pipefail", "-c"]

ENV GOSU_VERSION=1.17
RUN apk add --no-cache --virtual .gosu-deps \
		ca-certificates==20240705-r0 \
		dpkg==1.22.6-r1 \
		gnupg==2.4.5-r0 && \
	dpkgArch="$(dpkg --print-architecture | awk -F- '{ print $NF }')" && \
	wget -q -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$dpkgArch" && \
	wget -q -O /usr/local/bin/gosu.asc "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$dpkgArch.asc" && \
	GNUPGHOME="$(mktemp -d)" && \
	export GNUPGHOME && \
	gpg --batch --keyserver hkps://keys.openpgp.org --recv-keys B42F6819007F00F88E364FD4036A9C25BF357DD4 && \
	gpg --batch --verify /usr/local/bin/gosu.asc /usr/local/bin/gosu && \
	gpgconf --kill all && \
	rm -rf "$GNUPGHOME" /usr/local/bin/gosu.asc && \
	apk del --no-network .gosu-deps && \
	chmod +x /usr/local/bin/gosu

WORKDIR /app

RUN adduser -S -G nobody scyllaridae

RUN apk update && \
    apk add --no-cache \
      curl==8.11.1-r0 \
      bash==5.2.26-r0 \
      ca-certificates==20240705-r0 \
      openssl==3.3.2-r1

COPY . ./

RUN chown -R scyllaridae:nobody /app

RUN go mod download && \
  go build -o /app/scyllaridae && \
  go clean -cache -modcache && \
  ./ca-certs.sh

ENTRYPOINT ["/bin/bash"]
CMD ["/app/docker-entrypoint.sh"]
