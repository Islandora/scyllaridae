#!/usr/bin/env bash

set -eou pipefail

get_latest_version() {
    PACKAGE_NAME=$1
    # TODO: is there really no JSON API for this?
    PACKAGE_REPO_URL="https://pkgs.alpinelinux.org/package/v3.20/main/x86_64/$PACKAGE_NAME"
    PACKAGE_INFO=$(curl -s "$PACKAGE_REPO_URL")
    if echo "$PACKAGE_INFO" | grep -q "404 Page not found"; then
        PACKAGE_INFO=$(curl -s "https://pkgs.alpinelinux.org/package/v3.20/community/x86_64/$PACKAGE_NAME")
    fi
    LATEST_VERSION=$(echo "$PACKAGE_INFO" | grep -A 2 '<th class="header">Version</th>' | tail -n 1 | xargs)

    echo "$LATEST_VERSION"
}

find . -name 'Dockerfile' | while read -r DOCKERFILE; do
    echo "Checking $DOCKERFILE"
    grep -s '==' "$DOCKERFILE" > /dev/null || continue
    ggrep -soP '[a-zA-Z0-9_\-]+==[a-zA-Z0-9_\-\.]+' "$DOCKERFILE" | tr -d '\\' | while read -r PACKAGE; do
        echo -e "\tChecking $PACKAGE"
        PACKAGE_NAME=$(echo "$PACKAGE" | cut -d'=' -f1|awk '{print $1}')
        CURRENT_VERSION=$(echo "$PACKAGE" | cut -d'=' -f3|awk '{print $1}')
        LATEST_VERSION=$(get_latest_version "$PACKAGE_NAME")
        if [ "$LATEST_VERSION" = "" ]; then
          continue
        fi
        if [[ "$LATEST_VERSION" != "$CURRENT_VERSION" ]]; then
            echo -e "\t\tUpdating to $LATEST_VERSION"
            sed -E "s/($PACKAGE_NAME)==$CURRENT_VERSION/\1==$LATEST_VERSION/" "$DOCKERFILE" > "${DOCKERFILE}.bak"
            mv "${DOCKERFILE}.bak" "$DOCKERFILE"
        else
            echo -e "\t\t$PACKAGE_NAME is already up to date."
        fi
    done
done
