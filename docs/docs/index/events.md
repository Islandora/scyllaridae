If you've never written a microservice for Islandora before, you may not be familiar with exactly how events make their way to the microservice.

When you create/update/delete nodes or media from Drupal, Drupal emits an index event and that event is put on ActiveMQ's event queue.

The event will look something like this:

```
{
  "@context": "https://www.w3.org/ns/activitystreams",
  "actor": {
    "type": "Person",
    "id": "urn:uuid:b3f0a1ba-fd0c-4977-a123-3faf470374f2",
    "url": [
      {
        "name": "Canonical",
        "type": "Link",
        "href": "https://islandora.dev/user/1",
        "mediaType": "text/html",
        "rel": "canonical"
      }
    ]
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
  "target": "https://fcrepo.islandora.dev/fcrepo/rest/",
  "type": "Update",
  "summary": "Update a Node"
}
```

[islandora/alpaca](https://github.com/islandora/alpaca) is subscribed to ActiveMQ and when a message is added to the queue, alpaca reads the event and forwards the event to a microservice (alpaca's config associates a given queue with a given microservice as a 1:1 relationship).

Alpaca has some custom logic implemented for [triplestores](https://github.com/Islandora/Alpaca/blob/2.x/islandora-indexing-triplestore/src/main/java/ca/islandora/alpaca/indexing/triplestore/TriplestoreIndexer.java) and [fedora](https://github.com/Islandora/Alpaca/blob/2.x/islandora-indexing-fcrepo/src/main/java/ca/islandora/alpaca/indexing/fcrepo/FcrepoIndexer.java)
