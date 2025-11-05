#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] send_email.sh - Input: $INPUT" >&2
fi

email=$(echo "$INPUT" | jq -r '.email // "user@example.com"')
result=$(echo "$INPUT" | jq ".email_sent = true | .email_address = \"${email}\" | .step = \"notify_email\"")

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] send_email.sh - Output: $result" >&2
fi

echo "$result"

