#!/bin/bash

# Setup script for local PostgreSQL database
# This script will create the database and apply the schema

set -e

echo "Setting up local PostgreSQL for Callisto..."

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | xargs)
fi

# Database connection string
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"

# If no password was provided, try without one

echo "Creating database '${POSTGRES_DB}' if it doesn't exist..."
if [ -n "${POSTGRES_PASSWORD}" ]; then
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '${POSTGRES_DB}'" | grep -q 1 || \
            PGPASSWORD="$POSTGRES_PASSWORD" psql -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER -d postgres -c "CREATE DATABASE ${POSTGRES_DB};"
else
    psql -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '${POSTGRES_DB}'" | grep -q 1 || \
            psql -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER -d postgres -c "CREATE DATABASE ${POSTGRES_DB};"
fi

echo "Applying database schema..."
if [ -n "${POSTGRES_PASSWORD}" ]; then
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f database/schema/schema.sql
else
    psql -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f database/schema/schema.sql
fi

echo "✅ Database setup complete!"
echo ""
echo "Database connection details:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $POSTGRES_DB"
echo "  User: $POSTGRES_USER"
echo ""
echo "You can now start the services with: docker compose up -d"
