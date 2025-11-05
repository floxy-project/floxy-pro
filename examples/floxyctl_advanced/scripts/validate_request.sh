#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] validate_request.sh - Input: $INPUT" >&2
fi

data_type=$(echo "$INPUT" | jq -r '.data_type // "light"')
user_id=$(echo "$INPUT" | jq -r '.user_id // ""')

if [ -z "$user_id" ]; then
  echo "Error: user_id is required" >&2
  exit 1
fi

result=$(echo "$INPUT" | jq ".validated = true | .step = \"validate\" | .timestamp = $(date +%s)")

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] validate_request.sh - Output: $result" >&2
fi

echo "$result"

