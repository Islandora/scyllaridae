You can test your microservice by building it, running the container, and sending a file to it.


## Build

First, build your Dockerfile locally

```
cd path/to/your/Dockerfile/and/YML
docker build -t your-microservice:latest .
```

## Start
Now, start the docker container. Notice we're passing a `SKIP_JWT_VERIFY` environment variable. That tells scyllaridae to not require any JWT token verification.

```
docker run \
  --env SKIP_JWT_VERIFY=true \
  -p 8080:8080 \
  your-microservice:latest
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
