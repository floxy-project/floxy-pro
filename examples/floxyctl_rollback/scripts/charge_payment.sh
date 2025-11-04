#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] charge_payment.sh - Input: $INPUT" >&2
fi

amount=$(echo "$INPUT" | jq -r '.amount // 0')
if [ "$amount" = "0" ]; then
  echo "Error: amount is required" >&2
  exit 1
fi

should_fail=$(echo "$INPUT" | jq -r '.fail_charge // false')
if [ "$should_fail" = "true" ]; then
  echo "Error: payment charge failed" >&2
  exit 1
fi

result=$(echo "$INPUT" | jq '.payment_charged = true | .transaction_id = "TXN-98765" | .step = "charge_payment"')

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] charge_payment.sh - Output: $result" >&2
fi

echo "$result"
