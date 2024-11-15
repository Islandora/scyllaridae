ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM islandora/leptonica:alpine-3.20.2-leptonica-1.84.1-r0 AS leptonica
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

RUN --mount=type=bind,from=leptonica,source=/packages,target=/packages \
    --mount=type=bind,from=leptonica,source=/etc/apk/keys,target=/etc/apk/keys \
    apk update && \
    apk add --no-cache \
        /packages/leptonica-*.apk \
        tesseract-ocr==5.3.4-r0 \
        tesseract-ocr-data-eng==5.3.4-r0 \
        tesseract-ocr-data-fra==5.3.4-r0 \
        tesseract-ocr-data-spa==5.3.4-r0 \
        tesseract-ocr-data-ita==5.3.4-r0 \
        tesseract-ocr-data-por==5.3.4-r0 \
        tesseract-ocr-data-hin==5.3.4-r0 \
        tesseract-ocr-data-deu==5.3.4-r0 \
        tesseract-ocr-data-jpn==5.3.4-r0 \
        tesseract-ocr-data-rus==5.3.4-r0 \
        poppler-utils==24.02.0-r1

COPY scyllaridae.yml /app/scyllaridae.yml
