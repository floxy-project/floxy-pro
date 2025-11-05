#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] send_sms.sh - Input: $INPUT" >&2
fi

phone=$(echo "$INPUT" | jq -r '.phone // "+1234567890"')
result=$(echo "$INPUT" | jq ".sms_sent = true | .phone_number = \"${phone}\" | .step = \"notify_sms\"")

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] send_sms.sh - Output: $result" >&2
fi

echo "$result"

