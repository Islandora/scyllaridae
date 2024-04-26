ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

RUN apk update && \
    apk add leptonica-progs \
        tesseract-ocr \
        tesseract-ocr-eng \
        tesseract-ocr-fra \
        tesseract-ocr-spa \
        tesseract-ocr-ita \
        tesseract-ocr-por \
        tesseract-ocr-hin \
        tesseract-ocr-deu \
        tesseract-ocr-jpn \
        tesseract-ocr-rus \
        poppler-utils

COPY scyllaridae.yml /app/scyllaridae.yml
