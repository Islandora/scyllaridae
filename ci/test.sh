#!/usr/bin/env bash

set -eou pipefail

hash() {
  if command -v md5sum >/dev/null 2>&1; then
    md5sum "$@"
  else
    md5 "$@"
  fi
}

SERVICES=(
  "tesseract"
  "imagemagick"
  "crayfits"
  "ffmpeg"
  "whisper"
  "pandoc"
)
for SERVICE in "${SERVICES[@]}"; do
  URL="http://$SERVICE:8080/"
  echo "Testing $SERVICE at $URL"

  if [ "$SERVICE" == "crayfits" ]; then
    curl -s -o fits.xml \
        --header "Accept: application/xml" \
        --header "Apix-Ldp-Resource: https://preserve.lehigh.edu/_flysystem/fedora/2024-01/384659.pdf" \
        "$URL"
    # check the md5 of that file exists in the FITS XML
    grep c4b7c84671428767e3b0d9193c9c444b fits.xml | grep -q md5checksum && echo "FITS ran successfully"
    rm fits.xml
  elif [ "$SERVICE" == "ffmpeg" ]; then
    curl -s -o image.jpg \
        --header "X-Islandora-Args: -ss 00:00:45.000 -frames 1 -vf scale=720:-2" \
        --header "Accept: image/jpeg" \
        --header "Apix-Ldp-Resource: http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4" \
        "$URL"
    hash image.jpg | grep fe7dd57460dbaf50faa38affde54b694
    rm image.jpg
  elif [ "$SERVICE" == "imagemagick" ]; then
    curl -s -o image.png \
        --header "Accept: image/png" \
        --header "Apix-Ldp-Resource: https://preserve.lehigh.edu/_flysystem/fedora/2024-01/384659.pdf" \
        "$URL"
    file image.png | grep -q PNG && echo "PNG thumbnail created from PDF"
    rm image.png
  elif [ "$SERVICE" == "tesseract" ]; then
    curl -s -o ocr.txt \
        --header "Accept: text/plain" \
        --header "Apix-Ldp-Resource: https://preserve.lehigh.edu/sites/default/files/2023-12/285660.jpg" \
        "$URL"
    grep -q Pyrases ocr.txt || exit 1
    echo "Image OCR as expected"

    curl -s -o ocr.txt \
        --header "Accept: text/plain" \
        --header "Apix-Ldp-Resource: https://preserve.lehigh.edu/_flysystem/fedora/2024-01/384659.pdf" \
        "$URL"
    grep "One time I was ridin' along on the mule" ocr.txt || exit 1
    echo "PDF OCR as expected"
    rm ocr.txt
  elif [ "$SERVICE" == "whisper" ]; then
    curl -s -o vtt.txt \
        --header "Accept: text/plain" \
        --header "Apix-Ldp-Resource: https://preserve.lehigh.edu/sites/default/files/derivatives/hls/node/8157/11230.m3u8" \
        "$URL"
    grep -i "Lehigh Business Blog" vtt.txt || exit 1
    echo "VTT as expected"
    rm vtt.txt
  elif [ "$SERVICE" == "pandoc" ]; then
    curl -o result.tex \
      -H "Content-Type: text/markdown" \
      -H "Accept: application/x-latex" \
      --data-binary "@/fixtures/pandoc/input.md" \
      "$URL"

    if diff -u result.tex "fixtures/pandoc/output.tex" > diff_output.txt; then
      echo "Test Passed: Output matches expected."
    else
      echo "Test Failed: Differences found."
      cat diff_output.txt
      exit 1
    fi
  else
    echo "Unknown service"
    exit 1
  fi
done
