#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] generate_report.sh - Input: $INPUT" >&2
fi

user_id=$(echo "$INPUT" | jq -r '.user_id')
report_id="RPT-${user_id}-$(date +%s)"
result=$(echo "$INPUT" | jq ".report_generated = true | .report_id = \"${report_id}\" | .step = \"generate\"")

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] generate_report.sh - Output: $result" >&2
fi

echo "$result"

