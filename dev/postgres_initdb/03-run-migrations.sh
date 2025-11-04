#!/bin/bash
set -e

echo "Running migrations..."

if [ -n "$POSTGRES_MULTIPLE_DATABASES" ]; then
    for db in $(echo $POSTGRES_MULTIPLE_DATABASES | tr ',' ' '); do
        echo "Running migrations for database: $db"

        if [ -d "/migrations_pro" ] && [ "$(ls -A /migrations_pro)" ]; then
            for migration in $(ls /migrations_pro/*.up.sql 2>/dev/null | sort); do
                if [ -f "$migration" ]; then
                    echo "Applying migration: $(basename $migration)"
                    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$db" -f "$migration"
                fi
            done
            echo "Migrations completed successfully for $db"
        else
            echo "No migrations found in /migrations_pro directory"
        fi
    done
else
    echo "Running migrations for default database: $POSTGRES_DB"

    if [ -d "/migrations_pro" ] && [ "$(ls -A /migrations_pro)" ]; then
        for migration in $(ls /migrations_pro/*.up.sql 2>/dev/null | sort); do
            if [ -f "$migration" ]; then
                echo "Applying migration: $(basename $migration)"
                psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "$migration"
            fi
        done
        echo "Migrations completed successfully"
    else
        echo "No migrations found in /migrations_pro directory"
    fi
fi
