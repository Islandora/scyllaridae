To create your derivative microservice you need to define:

1. [A Dockerfile](#dockerfile): Libraries/binaries/etc stuffed into a Dockerfile (i.e. install the necessary software your microservice will require)
2. [A YML file](#scyllaridaes-yml-spec): how your microservice will run when an event is sent to it from alpaca

## Dockerfile

Define the `Dockerfile` to run your microservice.

You must have your Dockerfile use `islandora/scyllaridae:main` as the base so your command will get called properly when an event is emitted. Doing this ensures an http service will be running when the Dockerfile starts, and it will handle running your microservice when events are received from alpaca.

Your Dockerfile just needs to use the `apk` package manager to install the software your microservice will run (or build from source, or however best to install your dependencies).

### Example Dockerfile for Crayfits

```dockerfile
FROM islandora/scyllaridae:main

RUN apk update && \
    apk add --no-cache curl==8.11.1-r0

COPY scyllaridae.yml /app/scyllaridae.yml
```

## scyllaridae's YML spec

Now that your software is available in a docker image, you'll need to understand scyllaridae's YML spec that specifies how the software will run when an event is received by the servce. The spec needs to be put in a file called `scyllaridae.yml`.

One very important concept to understand up front is the Islandora/Drupal media/file you will be creating a derivative of will be streamed into your command using `stdin` and your command must print the file output (and no error text or other information) to `stdout`.

For the complete configuration specification, see the [Configuration Reference](../configuration.md).

The basic structure is as follows (with the default values shown):

```yaml
# boolean
# Indicates whether the authentication header (i.e. your Drupal JWT) should be forwarded
# when accessing your source media file
forwardAuth: true

# list of strings
# MIME type(s) allowed for processing.
# asterisks can be used to allow everything "*"
# or to allow certain groups e.g. image/*
allowedMimeTypes: []

# map[string]cmd
# key is the mimetype to perform a crafted command if the microservice needs special handling for a particular mime type
# the "default" keyword would apply to all mimetypes that get sent to the microservice
# default will only run if the mimetype does not have a special command set
#
# value is the command to run, which is a another map with two keys: cmd and args
# cmd is a string and is the executable that will be called
# args are the list of arguments to pass to cmd
cmdByMimeType: {}

# boolean
# Have cmdByMimeType base its execution on the mimetype for the destination file
# rather than the mimetype of the source file
mimeTypeFromDestination: false
```

A couple examples of the specs are

### All mimetypes handled the same

This example handles all mimetypes the same. This is what's used to send files to a Harvard FITS server running in the ISLE stack

```yaml
allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: "curl"
    args:
      - "-X"
      - "POST"
      - "-F"
      - "datafile=@-"
      - "http://fits:8080/fits/examine"
```

### By mimetype example

This example handles PDFs one way, and all images a different way. This can be used for a thumbnail generation microservice.

```yaml
allowedMimeTypes:
  - "application/pdf"
  - "image/*"
cmdByMimeType:
  "application/pdf":
    cmd: magick
    args:
      - "pdf:-[0]"
      - "%args"
      - "pdf:-"
  default:
    cmd: magick
    args:
      - "-"
      - "%args"
      - "image:-"
```

### Command arguments

Scyllaridae provides special variables that can be used in your `.cmdByMimeType[*].args`. These are automatically replaced with values from the incoming event:

- `%args` - Arguments passed via `X-Islandora-Args` header
- `%source-mime-ext` - Source file extension (e.g. `pdf`)
- `%destination-mime-ext` - Destination file extension (e.g. `jpg`)
- `%source-mime-pandoc` - Source MIME type in Pandoc format
- `%destination-mime-pandoc` - Destination MIME type in Pandoc format
- `%source-uri` - URL of the source file
- `%file-upload-uri` - Drupal URI where the derivative will be saved
- `%destination-uri` - URL where the derivative will be PUT
- `%canonical` - Canonical link to the node
- `%target` - Target value from the event

For the complete list of special variables and their usage, see the [Configuration Reference](../configuration.md#special-argument-variables)

### On stdin/stdout

If your command/software has flags to handle reading from stdin and writing to stdout, you can use those. Both examples above are using software (i.e. `curl` and `magick`) that allow reading from stdin and writing to stdout.

If your software does not have flags to handle this, you can just write a bash script to do this for you. For example a file `cmd.sh` could contain

```bash
#!/usr/bin/env bash

# take input from stdin and print to stdout

set -eou pipefail

input_temp=$(mktemp /tmp/libreoffice-input-XXXXXX)

# write stdin to a temp file
cat > "$input_temp"

# all stderr and stdout from the command is redirected to /dev/null
# since the libreoffice command will be writing to a file we don't
# want any command output being embedded in our derivative file
libreoffice --headless --convert-to pdf "$input_temp" > /dev/null 2>&1

PDF="/app/$(basename "$input_temp").pdf"

# print file to stdout
cat "$PDF"

# cleanup
rm "$input_temp" "$PDF"
```

Then scyllaridae.yml just calls the bash script instead of defining the cmd/args in the YML.

```yaml
allowedMimeTypes:
  - "application/msword"
  - "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
  - "application/vnd.ms-powerpoint"
  - "application/vnd.openxmlformats-officedocument.presentationml.presentation"
  - "application/vnd.ms-excel"
  - "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
cmdByMimeType:
  default:
    cmd: /app/cmd.sh
```
