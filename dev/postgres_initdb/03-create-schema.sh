#!/bin/bash
set -e

create_schema_if_not_exists() {
    local db=$1
    echo "Ensuring schema 'workflows' exists in database: $db"
    PGPASSWORD="$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$db" -c "CREATE SCHEMA IF NOT EXISTS workflows;" >/dev/null
}

if [ -n "$POSTGRES_MULTIPLE_DATABASES" ]; then
    for db in $(echo "$POSTGRES_MULTIPLE_DATABASES" | tr ',' ' '); do
        create_schema_if_not_exists "$db"
    done
else
    create_schema_if_not_exists "$POSTGRES_DB"
fi

echo "✅ Schema 'workflows' created successfully!"
