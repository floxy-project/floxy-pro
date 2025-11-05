#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] cleanup_temp_files.sh - Input: $INPUT" >&2
fi

result=$(echo "$INPUT" | jq ".temp_files_cleaned = true | .step = \"skip_report\"")

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] cleanup_temp_files.sh - Output: $result" >&2
fi

echo "$result"

