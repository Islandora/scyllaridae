ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae

# renovate: datasource=repology depName=alpine_3_20/openjdk17-jre
ENV OPENJDK_VERSION="17.0.13_p11-r0"
# renovate: datasource=repology depName=alpine_3_20/libreoffice
ENV LIBREOFFICE_VERSION="7.6.7.2-r0"
# renovate: datasource=repology depName=alpine_3_20/ttf-dejavu
ENV TTF_DEJAVU_VERSION="2.37-r5"

COPY scyllaridae.yml /app/
COPY cmd.sh /app/

RUN apk update && \
  apk add --no-cache \
    openjdk17-jre=="${OPENJDK_VERSION}" \
    libreoffice=="${LIBREOFFICE_VERSION}" \
    ttf-dejavu=="${TTF_DEJAVU_VERSION}"
