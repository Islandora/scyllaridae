ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae

COPY scyllaridae.yml /app/
COPY cmd.sh /app/

RUN apk update && \
  apk add --no-cache \
    openjdk17-jre==17.0.13_p11-r0 \
    libreoffice==7.6.7.2-r0 \
    ttf-dejavu==2.37-r5
