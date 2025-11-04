#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] filter.sh - Input: $INPUT" >&2
fi

value=$(echo "$INPUT" | jq -r '.value')
if [ "$value" -gt 50 ]; then
  result=$(echo "$INPUT" | jq '.filtered = true | .step = "filter"')
else
  result=$(echo "$INPUT" | jq '.filtered = false | .step = "filter" | .skipped = true')
fi

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] filter.sh - Output: $result" >&2
fi

echo "$result"
