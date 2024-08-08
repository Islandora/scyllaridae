#!/usr/bin/env bash

set -eou pipefail

COUNT=0
DOMAINS=(
  "preserve.lehigh.edu"
  "helloworld.letsencrypt.org"
  "www.libops.io"
  "sandbox.islandora.ca"
)
for DOMAIN in "${DOMAINS[@]}"; do
  echo $DOMAIN
  CERTS=$(openssl s_client -connect "$DOMAIN:443" -showcerts </dev/null 2>/dev/null | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p')
  while read -r CERT; do
    if [[ "$CERT" == *"BEGIN CERTIFICATE"* ]]; then
      FILENAME="/usr/local/share/ca-certificates/ca_$COUNT.crt"
      COUNT=$(( COUNT + 1 ))
      rm -f "$FILENAME"
    fi
    echo "$CERT" >> "$FILENAME"
  done <<< "$CERTS"
done

update-ca-certificates
