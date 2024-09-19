ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae
FROM ghcr.io/ggerganov/whisper.cpp:main-cuda

RUN bash ./models/download-ggml-model.sh base.en

COPY --from=scyllaridae /app /app
COPY scyllaridae.yml /app

RUN apt-get update && apt-get install -y gosu \
  && rm -rf /var/lib/apt/lists/* \
  && groupadd -r nobody \
  && useradd -r -g nobody scyllaridae \
  && chmod +x /app/docker-entrypoint.sh

ENTRYPOINT ["/app/docker-entrypoint.sh"]
