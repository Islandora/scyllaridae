You can test your microservice by building it, running the container, and sending a file to it.


## Build

First, build your Dockerfile locally

```
cd path/to/your/Dockerfile/and/YML
docker build -t my-microservice:latest .
```

## Start

Now, start the docker container. Notice we're passing a `SKIP_JWT_VERIFY` environment variable. That tells scyllaridae to not require any JWT token verification.

We're also setting the port to `8080`. If that port is already used on your host machine, you can change `PORT` to any available port number.

The log level is also set to `DEBUG` to aid in testing.

```
PORT=8080
docker run \
  --name microservice-test
  --env PORT=$PORT \
  --env LOG_LEVEL=DEBUG \
  --env SKIP_JWT_VERIFY=true \
  -p "$PORT:$PORT" \
  -d \
  my-microservice:latest
```

## Test

Upload a WAV file to your microservice and save it as `derivative.mp3`.

You obviously should change the mimetype/file/output to match what you're building, but this gives you an example of running your microservice before wiring it up

```
curl \
  -H "Content-Type: audio/x-wav" \
  -o derivative.mp3 \
  --data-binary "@output.wav" \
  http://localhost:8080
```

## Debug

Check your logs if any issues come up

```
docker logs microservice-test
```
