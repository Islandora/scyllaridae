ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

ARG TESSERACT_VERSION=5.3.4-r0

RUN apk update && \
    apk add --no-cache \
        leptonica-dev==1.83.1-r0 \
        tesseract-ocr==${TESSERACT_VERSION} \
        tesseract-ocr-data-eng==${TESSERACT_VERSION} \
        tesseract-ocr-data-fra==${TESSERACT_VERSION} \
        tesseract-ocr-data-spa==${TESSERACT_VERSION} \
        tesseract-ocr-data-ita==${TESSERACT_VERSION} \
        tesseract-ocr-data-por==${TESSERACT_VERSION} \
        tesseract-ocr-data-hin==${TESSERACT_VERSION} \
        tesseract-ocr-data-deu==${TESSERACT_VERSION} \
        tesseract-ocr-data-jpn==${TESSERACT_VERSION} \
        tesseract-ocr-data-rus==${TESSERACT_VERSION} \
        poppler-utils==23.10.0-r0

COPY scyllaridae.yml /app/scyllaridae.yml
