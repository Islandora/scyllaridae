# ocrpdf

Add OCR to PDF with no OCR

## Install

### Deploy microservice


#### docker-compose

Add the microservice to your docker compose

```
    ocrpdf-dev: &ocrpdf
        <<: [*dev, *common]
        image: lehighlts/scyllaridae-ocrpdf:main
        networks:
            default:
                aliases:
                    - ocrpdf
    ocrpdf-prod:
        <<: [*prod, *ocrpdf]
```

#### kubernetes

See [service/deployment manifest in scyllaridae repo](https://github.com/lehigh-university-libraries/scyllaridae/blob/main/ci/k8s/ocrpdf.yaml)


### Configure alpaca

You'll also need to add `ocrpdf` to `derivative.systems.installed` in your `alpaca.properties` by adding that string to the `ALPACA_DERIVATIVE_SYSTEMS` environment variable in your alpaca service.

```
ALPACA_DERIVATIVE_SYSTEMS=ocrpdf
```

You'll also need to define the service in alpaca.properties.tmpl

```
derivative.ocrpdf.enabled=true
derivative.ocrpdf.in.stream=queue:islandora-connector-ocrpdf
# this url may be different if deploying via kubernetes
derivative.ocrpdf.service.url=http://ocrpdf:8080
derivative.ocrpdf.concurrent-consumers=1
derivative.ocrpdf.max-concurrent-consumers=-1
derivative.ocrpdf.async-consumer=true
```
