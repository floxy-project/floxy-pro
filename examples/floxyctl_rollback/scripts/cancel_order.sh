#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] cancel_order.sh - Input: $INPUT" >&2
fi

order_id=$(echo "$INPUT" | jq -r '.order_id // "N/A"')
result=$(echo "$INPUT" | jq ".order_cancelled = true | .cancelled_order_id = \"${order_id}\" | .step = \"cancel_order\" | .reason = \"compensation\"")

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] cancel_order.sh - Output: $result" >&2
fi

echo "$result"
