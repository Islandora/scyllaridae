# Streaming Audio and Video Derivatives

Create an HLS streaming derivative for audio and video files

Requires [drupal/islandora_hls](https://www.drupal.org/project/islandora_hls) module running on your Islandora site to render the streaming derivative.

Also requires the parry microservice running in your ISLE stack

```
    parry-dev: &parry
        <<: [*dev, *common]
        image: lehighlts/scyllaridae-parry:main
        volumes:
            - ./certs/rootCA.pem:/app/ca.pem:r
        networks:
            default:
                aliases:
                    - parry
    hls-dev: &hls
        <<: [*dev, *common]
        image: lehighlts/scyllaridae-hls:main
        volumes:
            - ./certs/rootCA.pem:/app/ca.pem:r
        networks:
            default:
                aliases:
                    - hls
```
