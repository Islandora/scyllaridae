FROM python:3.14-alpine3.22@sha256:3a70fbae6ca02053139fe18c1041d4396c82b5c6e740d6fe1c24e7d90c22ac43

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
FROM nginx:1.30@sha256:55d1fb0944aafb8534b253e486c6cf57d3e0c785c4f6a7b3e9f02c5a080fb2ce
COPY --from=0 /docs/site /usr/share/nginx/html
