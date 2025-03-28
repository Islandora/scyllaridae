FROM python:3.13-alpine@sha256:323a717dc4a010fee21e3f1aac738ee10bb485de4e7593ce242b36ee48d6b352

RUN mkdir -p /docs

WORKDIR /docs

COPY requirements.txt .
RUN apk add --no-cache cairo \
  && pip install -r requirements.txt 

COPY . .
RUN mkdocs build

# serve the docs via nginx
FROM nginx:1.27@sha256:124b44bfc9ccd1f3cedf4b592d4d1e8bddb78b51ec2ed5056c52d3692baebc19
COPY --from=0 /docs/site /usr/share/nginx/html
