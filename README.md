# scyllaridae

Any command that takes stdin as input and streams its output to stdout can use scyllaridae.

## Adding a new microservice

### Create the microservice

Define the service's behavior in `scyllaridae.yml`.

You can specify which mimetypes the service can act on in `allowedMimeTypes`

And specify different commands for different mimetypes in `cmdByMimeType`, or set the default command to run for all mimetypes with the `default` key.

```yaml
allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: "curl"
    args:
      - "-X"
      - "POST"
      # read the source media file in from stdin
      - "-F"
      - "datafile=@-"
      # send the media file to FITS which will return the XML to stdout
      - "http://fits:8080/fits/examine"
```

<sup><sub>Here's another more [complex example YML](./scyllaridae.complex.yml).</sub></sup>


Define the `Dockerfile` to run your microservice. Your service will run the main `scyllaridae` program which is an http service configured by your `scyllaridae.yml`. You just need to install the binaries your yml specifies to ensure the command is in the container when it runs.

```dockerfile
FROM jcorall/scyllaridae:main

RUN apk update && \
    apk add --no-cache curl==8.5.0-r0

COPY scyllaridae.yml /app/scyllaridae.yml
```


### Deploy your new microservice

Update your [ISLE docker-compose.yml](https://github.com/Islandora-Devops/isle-site-template/blob/main/docker-compose.yml) to deploy the service's docker image defined above

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

This is spiritually a fork of the php/symfony implementation at [Islandora/Crayfish](https://github.com/Islandora/crayfish). The implementation of Crayfish was then generalized here to allow new microservices to just define a Dockerfile to install the proper binary/dependencies and a YML spec to execute the binary depending on the mimetype being processed. Hence the name of this service. [From Wikipedia](https://en.wikipedia.org/wiki/Slipper_lobster):

> Slipper lobsters are a family (Scyllaridae) of about 90 species of achelate crustaceans
