# scyllaridae

Any command that takes a file as input and prints a result as output can use scyllaridae.


## Basic overview

This service reads a file, and pipes the file's stream as `stdin` to a command. The `stdout` from that command is then returned as an HTTP response.

Both `GET` and `POST` requests are supported by any scyllaridea service.

`GET` supports Islandora's alpaca/event spec, which sends the URL of a file as an HTTP header `Apix-Ldp-Resource` and prints the result. e.g. to create a VTT file from an audio file:

```
$ curl -H "Apix-Ldp-Resource: https://github.com/ggerganov/whisper.cpp/raw/master/samples/jfk.wav" http://localhost:8080
WEBVTT

00:00:00.000 --> 00:00:11.000
 And so my fellow Americans, ask not what your country can do for you, ask what you can do for your country.
```

`POST` supports directly uploading a file to the service, being sure to include the mimetype of the file in the `Content-Type` HTTP header

```
$ curl \
  -H "Content-Type: audio/x-wav" \
  --data-binary "@output.wav" \
  http://localhost:8080/
WEBVTT

00:00:00.000 --> 00:00:02.960
 Lehigh University Library Technology.
```

You can see several example implementations using this framework in [examples](./examples). Some examples send the file's contents directly to `stdin` if the command supports reading from that stream e.g. [fits](./examples/fits/scyllaridae.yml). For other commands that do not support reading directly from stdin, and instead requiring specifying a file path on disk, a bash script is implemented to act as a wrapper around the command. e.g. [libreoffice](./examples/libreoffice/cmd.sh)

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

#### Kubernetes deployment

The demo kubernetes manifests in this repo uses the `ingress-nginx` controller to route requests from a single domain to the proper microservice.

##### Create a the TLS secret

```
kubectl create secret tls CHANGE-ME --key key.pem --cert cert.pem
```

#### Apply the kubernetes manifests

Now you can apply your kube manifests, being sure to replace `__DOMAIN__` and `__DOCKER_REPOSITORY__` with the proper valuyes

```
DOMAIN=CHANGE-ME.bar.com
DOCKER_REPOSITORY=lehighlts
KUBE_TLS_SECRET=CHANGE-ME
sed -e "s|__DOMAIN__|$DOMAIN|" \
    -e "s|__DOCKER_REPOSITORY__|$DOCKER_REPOSITORY|" \
    -e "s|__KUBE_TLS_SECRET__|$KUBE_TLS_SECRET|" \
    *.yaml \
| kubectl apply -f -
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
