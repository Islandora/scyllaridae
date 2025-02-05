## Drupal Actions/Context

Next, you need to define an action and context on when to execute your derivative

You'll want to read the Islandora docs around this: https://islandora.github.io/documentation/concepts/derivatives/

I typically try to find an existing action at `/admin/config/system/actions` to create the new action. I don't have great guidance around this as this setup seems a little less flexible than might be desired, but it typically only takes about five minutes to figure out the correct on. We should probably do a bit of work in the islandora drupal module to make this easier.

But the import thing is you'll want to remember what you set for the queue name (you'll want to create a new queue name). You'll need that value later on when configuring alpaca
To deploy your microservice, you need to have the service running in your docker compose deployment and make sure alpaca is aware of the service.

## docker-compose

Add the microservice to your docker compose, being sure to replace `YOUR-MICROSERVICE` with your actual microservce name

```
    YOUR-MICROSERVICE-dev: &YOUR-MICROSERVICE
        <<: [*dev, *common]
        image: lehighlts/scyllaridae-YOUR-MICROSERVICE:main
        networks:
            default:
                aliases:
                    - YOUR-MICROSERVICE
    YOUR-MICROSERVICE-prod:
        <<: [*prod, *YOUR-MICROSERVICE]
```

## Configure alpaca

You'll also need to add `YOUR-MICROSERVICE` to `derivative.systems.installed` in your `alpaca.properties` by adding that string to the `ALPACA_DERIVATIVE_SYSTEMS` environment variable in your alpaca service, being sure to replace `YOUR-MICROSERVICE` with your actual microservce name

```
ALPACA_DERIVATIVE_SYSTEMS=YOUR-MICROSERVICE
```

### override alpaca properties

You'll also need to define the service in alpaca.properties.tmpl. If running docker, and you haven't already overwritten the alpaca conf,g you'll want to grab the default value for `alpaca.properties.tmpl` from https://github.com/Islandora-Devops/isle-buildkit/blob/main/alpaca/rootfs/etc/confd/templates/alpaca.properties.tmpl and put that in your docker compose directory, and mount the file in your alpaca service using a volume mount

```
    alpaca-prod: &alpaca-prod
        <<: [*prod, *alpaca]
        image: ${ISLANDORA_REPOSITORY}/alpaca:${ISLANDORA_TAG}
        volumes:
            - ./conf/alpaca/alpaca.properties.tmpl:/etc/confd/templates/alpaca.properties.tmpl:r
```

### add your microservice to the queue

Now that you're overriding alpaca properties, you can add this to the bottom of alpaca.properties.tmpl, being sure to replace `islandora-connector-YOUR-MICROSERVICE` with your queue name from your Drupal action and `YOUR-MICROSERVICE` with your actual microservce

```
derivative.YOUR-MICROSERVICE.enabled=true
derivative.YOUR-MICROSERVICE.in.stream=queue:islandora-connector-YOUR-MICROSERVICE
# this url may be different if deploying via kubernetes
derivative.YOUR-MICROSERVICE.service.url=http://YOUR-MICROSERVICE:8080
derivative.YOUR-MICROSERVICE.concurrent-consumers=1
derivative.YOUR-MICROSERVICE.max-concurrent-consumers=-1
derivative.YOUR-MICROSERVICE.async-consumer=true
```
