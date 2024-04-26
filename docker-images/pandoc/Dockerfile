ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM pandoc/minimal:3.1.1 AS pandoc
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

COPY --from=pandoc /usr/local/bin/pandoc /usr/local/bin/pandoc
COPY scyllaridae.yml .
ENTRYPOINT ["/app/scyllaridae"]
