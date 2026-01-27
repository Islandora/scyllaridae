FROM golang:1.25-alpine3.22@sha256:d9c983d2ac66c3f43208dfb6b092dd1296baa058766e3aa88a1b233adeb416c1

SHELL ["/bin/ash", "-o", "pipefail", "-c"]

ARG \
  # renovate: datasource=repology depName=alpine_3_22/ca-certificates
  CA_CERTIFICATES_VERSION=20250911-r0 \
  # renovate: datasource=repology depName=alpine_3_22/dpkg
  DPKG_VERSION=1.22.15-r0 \
  # renovate: datasource=repology depName=alpine_3_22/gnupg
  GNUPG_VERSION=2.4.9-r0 \
  # renovate: datasource=repology depName=alpine_3_22/curl
  CURL_VERSION=8.14.1-r2 \
  # renovate: datasource=repology depName=alpine_3_22/bash
  BASH_VERSION=5.2.37-r0 \
  # renovate: datasource=repology depName=alpine_3_22/openssl
  OPENSSL_VERSION=3.5.5-r0 \
  # renovate: datasource=github-releases depName=gosu packageName=tianon/gosu
  GOSU_VERSION=1.19

# install gosu
RUN apk add --no-cache --virtual .gosu-deps \
    ca-certificates=="${CA_CERTIFICATES_VERSION}" \
    dpkg=="${DPKG_VERSION}" \
    gnupg=="${GNUPG_VERSION}" && \
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

ENV \
  SCYLLARIDAE_LOG_LEVEL=INFO \
  SCYLLARIDAE_PORT=8080 \
  SCYLLARIDAE_YML_PATH="/app/scyllaridae.yml"

RUN adduser -S -G nobody scyllaridae

RUN apk update && \
    apk add --no-cache \
      curl=="${CURL_VERSION}" \
      bash=="${BASH_VERSION}" \
      ca-certificates=="${CA_CERTIFICATES_VERSION}" \
      openssl=="${OPENSSL_VERSION}"

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . ./

RUN chown -R scyllaridae:nobody /app /tmp

RUN go build -o /app/scyllaridae && \
  go clean -cache -modcache

ENTRYPOINT ["/bin/bash"]
CMD ["/app/docker-entrypoint.sh"]
