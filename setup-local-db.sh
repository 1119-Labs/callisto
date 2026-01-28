#!/bin/bash

# Setup script for local PostgreSQL database
# This script will create the database and apply the schema

set -e

echo "Setting up local PostgreSQL for Callisto..."

# Load environment variables
export $(cat .env | xargs)

# Database connection string
DB_HOST="localhost"
DB_PORT="5432"

echo "Creating database '${POSTGRES_DB}' if it doesn't exist..."
psql -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER -tc "SELECT 1 FROM pg_database WHERE datname = '${POSTGRES_DB}'" | grep -q 1 || \
    psql -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER -c "CREATE DATABASE ${POSTGRES_DB};"

echo "Applying database schema..."
psql -h $DB_HOST -p $DB_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f database/schema/schema.sql

echo "✅ Database setup complete!"
echo ""
echo "Database connection details:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $POSTGRES_DB"
echo "  User: $POSTGRES_USER"
echo ""
echo "You can now start the services with: docker compose up -d"
