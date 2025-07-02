ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

ENV \
  FITS_URI=http://fits:8080/fits/examine

COPY scyllaridae.yml /app/scyllaridae.yml
