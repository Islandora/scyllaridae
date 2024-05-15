ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG} AS scyllaridae

RUN apk update && \
    apk add --no-cache \
      jq==1.7.1-r0 \
      texlive==20230506.66984-r3 \
      texlive-xetex==20230506.66984-r3 \
      ghostscript==10.02.1-r0 \
      html2text==2.2.3-r1 \
      fontconfig==2.14.2-r4 \
      font-bakoma==1-r1 \
      font-dejavu==2.37-r5 \
      font-noto==23.7.1-r0 \
      font-noto-extra==23.7.1-r0 && \
    fc-cache -f -v

COPY . /app/

ENTRYPOINT ["/app/scyllaridae"]
