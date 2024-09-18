ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae
FROM ghcr.io/ggerganov/whisper.cpp:main-cuda

RUN bash ./models/download-ggml-model.sh base.en

COPY --from=scyllaridae /app /app
COPY scyllaridae.yml /app

ENTRYPOINT ["/app/docker-entrypoint.sh"]
