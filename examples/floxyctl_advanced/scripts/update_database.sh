#!/bin/bash

set -e

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] update_database.sh - Input: $INPUT" >&2
fi

user_id=$(echo "$INPUT" | jq -r '.user_id')
result=$(echo "$INPUT" | jq ".database_updated = true | .db_record_id = \"DB-${user_id}\" | .step = \"update_db\"")

if [ "$FLOXY_DEBUG" = "true" ]; then
  echo "[DEBUG] update_database.sh - Output: $result" >&2
fi

echo "$result"

