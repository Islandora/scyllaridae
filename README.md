# scyllaridae

Any command that takes stdin as input and streams its output to stdout can use scyllaridae.

## Adding a new microservice

### Create the microservice

`Dockerfile`
```dockerfile
ARG TAG=main
ARG DOCKER_REPOSITORY=local
FROM ${DOCKER_REPOSITORY}/scyllaridae:${TAG}

RUN apk update && \
    apk add --no-cache curl

COPY scyllaridae.yml /app/scyllaridae.yml
```

`scyllaridae.yml`
```yaml
allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: "curl"
    args:
      - "http://fits:8080/fits/examine"
      - "-X"
      - "POST"
      - "-F"
      - "datafile=@-"
```

### Deploy your new microservice

Update your ISLE docker-compose.yml to deploy the service's docker image defined above

```yaml
    fits-dev: &fits
        <<: [*dev, *common]
        image: ${DOCKER_REPOSITORY}/scyllaridae-fits:main
        networks:
            default:
                aliases:
                    - fits
    fits-prod: &fits-prod
        <<: [*prod, *fits]
```

### Configure alpaca and Drupal

Until we define a subscription spec for Islandora Events in this repo, you'll also need to:

3. Update alpaca.properties
```
derivative.QUEUE-NAME.enabled=true
derivative.QUEUE-NAME.in.stream=queue:islandora-connector-QUEUE-NAME
derivative.QUEUE-NAME.service.url=http://QUEUE-NAME:8080/
derivative.QUEUE-NAME.concurrent-consumers=-1
derivative.QUEUE-NAME.max-concurrent-consumers=-1
derivative.QUEUE-NAME.async-consumer=true
```
4. Add an action in Islandora with your `islandora-connector-QUEUE-NAME` specified
5. Add a context to trigger the action you created in step 4

## Attribution

This is spiritually a fork of the php/symfony implementation at https://github.com/Islandora/crayfish, generalized to allow new microservices to just define a Dockerfile to install the proper binary/depencies and a YML spec to execute the binary depending on the mimetype being processed.
