FROM python:3.14-alpine3.22@sha256:9509532e9f46143926b7b87ce87dca4ab914adb8eb21dcc225ea09c6912a51f0

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
FROM nginx:1.29@sha256:c881927c4077710ac4b1da63b83aa163937fb47457950c267d92f7e4dedf4aec
COPY --from=0 /docs/site /usr/share/nginx/html
