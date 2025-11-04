#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] create_order.sh - Input: $INPUT" >&2
fi

result=$(echo "$INPUT" | jq '.order_id = "ORD-12345" | .order_created = true | .step = "create_order"')

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] create_order.sh - Output: $result" >&2
fi

echo "$result"
