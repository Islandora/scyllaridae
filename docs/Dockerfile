FROM python:3.14-alpine3.22@sha256:355522f4e6b640e9255e6b6830ef52f32ece54df43db93f61101f48c5d4c6658

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
FROM nginx:1.30@sha256:1a9ab83e2892b75773978e8d91f42a7a2a0d8bb704959a51ff17c0377481973d
COPY --from=0 /docs/site /usr/share/nginx/html
