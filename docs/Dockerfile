FROM python:3.14-alpine3.22@sha256:6b91e66ab2a880ce9ca5a1b91c70f45963ff71ff68268df056336e1a657d5efd

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
FROM nginx:1.31@sha256:b34848eff6db786b6b1282d3a9c3fd0b5563dfb6d261df4923378b419e0d24f0
COPY --from=0 /docs/site /usr/share/nginx/html
