#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] transform.sh - Input: $INPUT" >&2
fi

result=$(echo "$INPUT" | jq '.transformed = true | .value = (.value * 3) | .step = "transform"')

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] transform.sh - Output: $result" >&2
fi

echo "$result"

