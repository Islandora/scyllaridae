ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

# renovate: datasource=repology depName=alpine_3_20/qpdf
ENV QPDF_VERSION="11.9.1-r0"

RUN apk update && \
    apk add --no-cache \
        qpdf=="${QPDF_VERSION}"

COPY . /app
