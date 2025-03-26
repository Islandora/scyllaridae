# scyllaridae crayfits

An alternative to [Islandora Crayfits](https://github.com/Islandora/Crayfish/tree/4.x/CrayFits)

This service's default [/app/scyllaridae.yml](./scyllaridae.yml) expects a Harvard FITS service available at `http://fits:8080/fits/examine`.

Using [isle-site-template](https://github.com/islandora-devops/isle-site-template) you could replace the existing `crayfits-dev` service with this one with one of the following two configs:

## Config (preferred)

If you have [drupal/islandora_jwks](https://www.drupal.org/project/islandora_jwks) installed on your Islandora Drupal site, you could set the following docker compose config

```yaml
    crayfits-dev: &crayfits
        <<: [*dev, *common]
        image: lehighlts/scyllaridae-fits:main
        environment:
          JWKS_URI: https://${DOMAIN}/oauth/discovery/keys
        volumes:
            - ./certs/rootCA.pem:/app/ca.pem:r
        networks:
            default:
                aliases:
                    - crayfits
```

This config requires a JWT signed by your Islandora site in order to generate a FITS derivative.

## Config (insecure)

If you do not have [drupal/islandora_jwks](https://www.drupal.org/project/islandora_jwks) installed on your Islandora Drupal site, you can avoid JWT verificaiton with the following docker compose config

```yaml
    crayfits-dev: &crayfits
        <<: [*dev, *common]
        image: lehighlts/scyllaridae-fits:main
        environment:
          SKIP_JWT_VERIFY: "true"
        volumes:
            - ./certs/rootCA.pem:/app/ca.pem:r
        networks:
            default:
                aliases:
                    - crayfits
```

This config allows anyone with network access to your docker service (by default only on your local machine) to generate a FITS derivative.
