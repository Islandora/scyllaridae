#!/usr/bin/env bash

set -eou pipefail

# Output SCYLLARIDAE_AUTH status to stderr for verification
if [ "${SCYLLARIDAE_AUTH+x}" = "x" ]; then
	echo "SCYLLARIDAE_AUTH=absent" >&2
else
	echo "SCYLLARIDAE_AUTH=present" >&2
fi

cat
