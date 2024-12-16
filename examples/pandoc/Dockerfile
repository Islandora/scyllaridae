ARG TAG=main
ARG DOCKER_REPOSITORY=lehighlts
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae
FROM pandoc/latex:3.5.0-ubuntu AS pandoc

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl=8.5.0-2ubuntu10.6 \
        bash=5.2.21-2ubuntu4 \
        gosu=1.17-1ubuntu0.24.04.2 \
        jq=1.7.1-3build1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=scyllaridae /app/scyllaridae /app/scyllaridae
COPY --from=scyllaridae /app/docker-entrypoint.sh /app/docker-entrypoint.sh
COPY scyllaridae.yml .

RUN groupadd -r nobody && \
    useradd -r -g nobody scyllaridae

ENTRYPOINT ["/bin/bash"]
CMD ["/app/docker-entrypoint.sh"]
