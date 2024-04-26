ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae

FROM pandoc/minimal:3.1.1
WORKDIR /app
COPY --from=scyllaridae /app/scyllaridae .
COPY scyllaridae.yml .
ENTRYPOINT ["/app/scyllaridae"]
