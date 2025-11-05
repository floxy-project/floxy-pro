#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] process_data_heavy.sh - Input: $INPUT" >&2
fi

sleep 1

result=$(echo "$INPUT" | jq '.processed = true | .processing_type = "heavy" | .processing_time_ms = 1000 | .step = "process_heavy"')

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] process_data_heavy.sh - Output: $result" >&2
fi

echo "$result"

