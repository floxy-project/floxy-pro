#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] send_notification.sh - Input: $INPUT" >&2
fi

result=$(echo "$INPUT" | jq '.notification_sent = true | .step = "send_notification"')

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] send_notification.sh - Output: $result" >&2
fi

echo "$result"
