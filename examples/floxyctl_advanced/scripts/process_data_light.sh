#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] process_data_light.sh - Input: $INPUT" >&2
fi

result=$(echo "$INPUT" | jq '.processed = true | .processing_type = "light" | .processing_time_ms = 100 | .step = "process_light"')

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] process_data_light.sh - Output: $result" >&2
fi

echo "$result"

