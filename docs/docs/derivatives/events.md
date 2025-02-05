If you've never written a microservice for Islandora before, you may not be familiar with exactly how events make their way to the microservice.

When you add a media entity in Drupal, that event is emitted from Drupal and put on ActiveMQ's event queue based on the Action settings.

The event will look something like this:

```
{
  "actor": {
    "id": "urn:uuid:01abcdef-2345-6789-abcd-ef0123456789"
  },
  "object": {
    "id": "urn:uuid:abcdef01-2345-6789-abcd-ef0123456789",
    "url": [
      {
        "name": "Canonical",
        "type": "Link",
        "href": "https://islandora.dev/node/1",
        "mediaType": "text/html",
        "rel": "canonical"
      },
      {
        "name": "JSON",
        "type": "Link",
        "href": "https://islandora.dev/node/1?_format=json",
        "mediaType": "application/json",
        "rel": "alternate"
      },
      {
        "name": "JSONLD",
        "type": "Link",
        "href": "https://islandora.dev/node/1?_format=jsonld",
        "mediaType": "application/ld+json",
        "rel": "alternate"
      }
    ],
    "isNewVersion": true
  },
  "attachment": {
    "type": "Object",
    "content": {
      "mimetype": "image/jpeg",
      "args": "-ss 00:00:03.000 -frames 1 -vf scale=100:-2",
      "source_uri": "https://example.com/path/to/file.mp4",
      "destination_uri": "https://example.com/node/1/media/image/2",
      "file_upload_uri": "private://2024-03/thumbnail.jpg"
    },
    "mediaType": "application/json"
  },
  "type": "Activity",
  "summary": "Generate Derivative"
}

```

[islandora/alpaca](https://github.com/islandora/alpaca) is subscribed to ActiveMQ and when a message is added to the queue, alpaca reads the event and forwards the event to a microservice (alpaca's config associates a given queue with a given microservice as a 1:1 relationship).

However, alpaca does not send the entire event message. It only sends [three attributes from the event](https://github.com/Islandora/Alpaca/blob/ef738a254c52f6eb6acce2ac1728f86175f72a7a/islandora-connector-derivative/src/main/java/ca/islandora/alpaca/connector/derivative/DerivativeConnector.java#L112-L114), and they are sent as HTTP headers. If the islandora event JSON was in a file `event.json` that command would look like:

```
curl \
  --header "Authorizaton: bearer xyz" \ # authorization is embeded in the event stream
  --header "Accept: $(jq .attachment.content.mimetype event.json)" \
  --header "X-Islandora-Args: $(jq .attachment.content.args event.json)" \
  --header "Apix-Ldp-Resource: $(jq .attachment.content.sourceUri event.json)" \
  http://example.com/your/micro/service
```

The microservice then reads those headers, downloads the file represented in the URI found in the `Apix-Ldp-Resource` header, sends the file to a command that generates a derivate of the file, and prints the derivative file contents to stdout. Alpaca then handles uploading that derivative to Drupal.
