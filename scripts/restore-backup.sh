#!/bin/bash
# =============================================================================
# Database Restore Script
# =============================================================================
# Restores a PostgreSQL database from a backup file.
#
# Usage:
#   ./restore-backup.sh <backup-file>
#   ./restore-backup.sh backups/lukaut_20241215_020000.dump
# =============================================================================

set -euo pipefail

if [ $# -ne 1 ]; then
  echo "Usage: $0 <backup-file>"
  echo "Example: $0 backups/lukaut_20241215_020000.dump"
  exit 1
fi

BACKUP_FILE="$1"

if [ ! -f "$BACKUP_FILE" ]; then
  echo "Error: Backup file not found: $BACKUP_FILE"
  exit 1
fi

echo "=============================================="
echo "Database Restore"
echo "=============================================="
echo "Backup file: $BACKUP_FILE"
echo ""
echo "WARNING: This will OVERWRITE the current database!"
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
  echo "Restore cancelled."
  exit 0
fi

# Get database credentials from environment
if [ -f .env.production ]; then
  export $(cat .env.production | grep -v '^#' | xargs)
fi

POSTGRES_USER="${POSTGRES_USER:-lukaut}"
POSTGRES_DB="${POSTGRES_DB:-lukaut}"

echo ""
echo "Stopping application..."
docker compose -f docker-compose.prod.yml stop app

echo ""
echo "Restoring database..."
docker compose -f docker-compose.prod.yml exec -T db pg_restore \
  --clean \
  --if-exists \
  --no-owner \
  --no-privileges \
  -U "$POSTGRES_USER" \
  -d "$POSTGRES_DB" \
  < "$BACKUP_FILE"

echo ""
echo "Starting application..."
docker compose -f docker-compose.prod.yml start app

echo ""
echo "Waiting for application to be ready..."
sleep 10

# Health check
if curl -sf http://localhost:8080/health > /dev/null; then
  echo "Restore complete! Application is healthy."
else
  echo "Warning: Application health check failed. Please investigate."
  exit 1
fi
