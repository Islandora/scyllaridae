FROM python:3.13-alpine@sha256:18159b2be11db91f84b8f8f655cd860f805dbd9e49a583ddaac8ab39bf4fe1a7

RUN mkdir -p /docs

WORKDIR /docs

COPY requirements.txt .
RUN apk add --no-cache cairo \
  && pip install -r requirements.txt 

COPY . .
RUN mkdocs build

# serve the docs via nginx
FROM nginx:1.27@sha256:5ed8fcc66f4ed123c1b2560ed708dc148755b6e4cbd8b943fab094f2c6bfa91e
COPY --from=0 /docs/site /usr/share/nginx/html
