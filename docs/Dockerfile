FROM python:3.13-alpine@sha256:bb2c06f24622d10187d0884b5b0a66426a9c8511c344492ed61b5d382bd6018c

RUN mkdir -p /docs

WORKDIR /docs

COPY requirements.txt .
RUN apk add --no-cache cairo \
  && pip install -r requirements.txt 

COPY . .
RUN mkdocs build

# serve the docs via nginx
FROM nginx:1.27@sha256:91734281c0ebfc6f1aea979cffeed5079cfe786228a71cc6f1f46a228cde6e34
COPY --from=0 /docs/site /usr/share/nginx/html
