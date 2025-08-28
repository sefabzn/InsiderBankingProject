#!/bin/bash
set -e

echo "Initializing PostgreSQL replica..."

# Wait for primary to be ready
echo "Waiting for primary database to be ready..."
until pg_isready -h db -p 5432 -U postgres; do
  echo "Waiting for primary database..."
  sleep 2
done

echo "Primary database is ready. Setting up replication..."

# Create replication slot on primary
echo "Creating replication slot..."
psql -h db -U postgres -c "SELECT pg_create_physical_replication_slot('replica_slot');" || true

echo "Replica initialization complete. Replication will start on next restart."
