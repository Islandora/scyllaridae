# scyllaridae

Any command that takes a file as input and prints a result as output can use scyllaridae.

## Basic overview

This service reads a file, pipes the file's stream as `stdin` to a command, and returns the `stdout` from that command as as an HTTP response.

Both `GET` and `POST` requests are supported by any scyllaridea service.

`GET` supports Islandora's alpaca/event spec, which sends the URL of a file as an HTTP header `Apix-Ldp-Resource` and prints the result. e.g. to create a VTT file from an audio file:

```
$ curl \
  -H "Apix-Ldp-Resource: https://github.com/ggerganov/whisper.cpp/raw/master/samples/jfk.wav" \
  http://localhost:8080
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
