#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] refund_payment.sh - Input: $INPUT" >&2
fi

transaction_id=$(echo "$INPUT" | jq -r '.transaction_id // "N/A"')
result=$(echo "$INPUT" | jq ".payment_refunded = true | .refund_transaction_id = \"REFUND-${transaction_id}\" | .step = \"refund_payment\" | .reason = \"compensation\"")

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] refund_payment.sh - Output: $result" >&2
fi

echo "$result"
