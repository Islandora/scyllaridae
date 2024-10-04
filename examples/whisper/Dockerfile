ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae
FROM ghcr.io/ggerganov/whisper.cpp:main-cuda

COPY --from=scyllaridae /app /app
COPY scyllaridae.yml /app
COPY cmd.sh /app

RUN bash ./models/download-ggml-model.sh medium.en

RUN apt-get update && apt-get install -y gosu=1.14-1 --no-install-recommends \
  && rm -rf /var/lib/apt/lists/* \
  && groupadd -r nobody \
  && useradd -r -g nobody scyllaridae \
  && chmod +x /app/docker-entrypoint.sh /app/cmd.sh \
  && chown scyllaridae /app/models

ENTRYPOINT ["/app/docker-entrypoint.sh"]
