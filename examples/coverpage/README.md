# PDF Coverpage

Add a cover page to PDFs uploaded to your repository.

## How it works

- [Configure scyllaridae](scyllaridae.yml) to subscribe to a queue on ActiveMQ
- Create a Drupal action to generate a coverpage when a PDF is uploaded in Islandora, sending the event to the `islandora-pdf-coverpage` queue (which is set in [scyllaridae.yml](scyllaridae.yml))
- Use [a LaTex file](./coverpage.tex) as the template for the coverpage
- Add metadata from the node to the coverpage, using pandoc to convert the HTML to LaTeX
  - Requires [the Islandora CSL module](https://www.drupal.org/project/islandora_csl) to add a `citation` field to your node `?_format=json` output
- Have `xelatex` convert the coverpage TeX to PDF
- Then use Ghostscript to add the cover page to the original PDF

## Install

Add the microservice to your ISLE stack

```
    coverpage-dev: &coverpage
        <<: [*dev, *common]
        image: ${DOCKER_REPOSITORY}/scyllaridae-coverpage:main
        volumes:
            # you can customize the coverpage template and the logo like so
            - ./path/to/your/logo.png:/app/logo.png:r
            - ./path/to/your/coverpage-template.tex:/app/coverpage.tex:r
        networks:
            default:
                aliases:
                    - coverpage
    coverpage-prod: &coverpage-prod
        <<: [*prod, *coverpage]
```

## TODO

- [ ] Make logo, verbiage, title configurable. Though you could just add `volumes` to your docker to overwrite `/app/logo.png` and `/app/coverpage.tex`
