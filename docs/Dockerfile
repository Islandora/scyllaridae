FROM python:3.14-alpine3.22@sha256:17a4e595ce2b9193166da00f2e80e3c07ed66be79822d451ffedc29ab8a2e1b4

ARG \
  # renovate: datasource=repology depName=alpine_3_22/cairo
  CAIRO_VERSION=1.18.4-r0

RUN mkdir -p /docs

WORKDIR /docs

COPY requirements.txt .
RUN apk add --no-cache cairo==${CAIRO_VERSION} \
  && pip install  --no-cache-dir -r requirements.txt 

COPY . .
RUN mkdocs build

# serve the docs via nginx
FROM nginx:1.30@sha256:0c3a604353135ac13f3a991e2a98c9958af7b8c0b7251ce1590e0c5ff5e3692c
COPY --from=0 /docs/site /usr/share/nginx/html
