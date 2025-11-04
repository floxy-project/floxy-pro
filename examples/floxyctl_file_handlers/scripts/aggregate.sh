#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] aggregate.sh - Input: $INPUT" >&2
fi

result=$(echo "$INPUT" | jq '.aggregated = true | .step = "aggregate" | .completed = true')

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] aggregate.sh - Output: $result" >&2
fi

echo "$result"
