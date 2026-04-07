#!/usr/bin/env sh
set -eu

# Deletes legacy claim attachment rows that predate Cloudinary migration.
# Safe: only removes rows missing Cloudinary metadata.

if [ ! -f "./compose-docker.sh" ]; then
  echo "Run from the backend repo root (where compose-docker.sh exists)."
  exit 1
fi

echo "Counting legacy claim attachments (missing cloudinary_public_id)..."
./compose-docker.sh exec -T postgres psql -U "${DB_USER:-postgres}" -d "${DB_NAME:-shukisha}" -c \
  "SELECT COUNT(*) AS legacy_rows FROM parcel_claim_attachments WHERE COALESCE(cloudinary_public_id, '') = '' OR COALESCE(cloudinary_resource_type, '') = '' OR COALESCE(cloudinary_version, 0) = 0;"

echo ""
echo "Deleting legacy claim attachments..."
./compose-docker.sh exec -T postgres psql -U "${DB_USER:-postgres}" -d "${DB_NAME:-shukisha}" -c \
  "BEGIN;
   DELETE FROM parcel_claim_attachments
   WHERE COALESCE(cloudinary_public_id, '') = '' OR COALESCE(cloudinary_resource_type, '') = '' OR COALESCE(cloudinary_version, 0) = 0;
   COMMIT;"

echo ""
echo "Done."

