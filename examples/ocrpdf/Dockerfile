ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae-imagemagick:${TAG} AS scyllaridae

RUN apk update && \
  apk add --no-cache \
    ghostscript==10.04.0-r0 \
    jq==1.7.1-r0 \
    leptonica-dev==1.84.1-r0 \
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

COPY . /app

ENTRYPOINT ["/app/docker-entrypoint.sh"]
