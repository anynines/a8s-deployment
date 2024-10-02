# PostgreSQL Version Migration Script

This script facilitates the migration of a PostgreSQL database between different versions within a Kubernetes environment. It addresses a specific limitation of the a8s framework components, which cannot perform backup and restore operations between different PostgreSQL instances.

The following provides a workaround for this limitation, allowing for migration between different PostgreSQL versions.

## Prerequisites

- Kubernetes cluster with kubectl configured
- Source and destination PostgreSQL instances (of different versions) running as pods in the cluster
- Sufficient permissions to execute kubectl commands and access PostgreSQL instances

## Usage

```bash
./migrate_postgresql_versions.sh.sh <SOURCE_INSTANCE> <DESTINATION_INSTANCE> [DB_NAME] [DB_USER]
```

### Parameters

- `<SOURCE_INSTANCE>`: The name of the source PostgreSQL pod (older version)
- `<DESTINATION_INSTANCE>`: The name of the destination PostgreSQL pod (newer version)
- `[DB_NAME]`: The name of the database to migrate (optional, defaults to 'a9s_apps_default_db')
- `[DB_USER]`: The PostgreSQL user to use for operations (optional, defaults to 'postgres')

## Script Workflow

1. Validates input parameters and checks for the existence of source and destination instances.
2. Creates a backup of the specified database on the source instance (older version).
3. Copies the backup file to the local machine.
4. Checks if the destination database exists in the newer version instance, creates it if necessary.
5. Copies the backup file to the destination instance.
6. Restores the backup on the destination instance (newer version).
7. Verifies the data restoration by listing tables in the destination database.

## Error Handling

The script includes error handling to catch and report issues during the migration process. If an error occurs, the script will display an error message and exit.

## Cleanup

The script automatically removes the temporary backup file from the local machine upon completion or in case of an error.

## Example

To migrate the 'myapp' database from PostgreSQL 12 to PostgreSQL 14:

```bash
./migrate_postgres_version.sh postgres-12-pod postgres-14-pod a9s_apps_default_db postgres
```

## Important Notes

- This script is specifically designed to work around the limitations of the a8s framework components in migrating data between different PostgreSQL versions.
- Ensure that you have tested the migration process in a non-production environment before applying it to production databases.
- Always have a backup of your data before performing any migration.
- The script assumes that the PostgreSQL instances are running in containers named 'postgres' within their respective pods.

migrate_postgresql_versions.sh

```bash
#!/bin/bash

set -e  # Exit immediately if a command exits with a non-zero status

# Check if required arguments are provided
if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <SOURCE_INSTANCE> <DESTINATION_INSTANCE> [DB_NAME] [DB_USER]"
    exit 1
fi

SOURCE_INSTANCE="$1"
DESTINATION_INSTANCE="$2"
DB_NAME="${3:-a9s_apps_default_db}"  # Default to 'a9s_apps_default_db' if not provided
DB_USER="${4:-postgres}"  # Default to 'postgres' if not provided
BACKUP_FILE="pg_backup_$(date +%Y%m%d_%H%M%S).sql"

# Function to execute PostgreSQL commands
exec_psql() {
    local instance="$1"
    local command="$2"
    kubectl exec "$instance" --container postgres -- psql -U "$DB_USER" -d "$DB_NAME" -c "$command"
}

# Function to handle errors
handle_error() {
    echo "Error: $1" >&2
    exit 1
}

# Trap to clean up on script exit
trap 'rm -f ./"$BACKUP_FILE"' EXIT

# Check if source instance exists and is accessible
echo "Checking source instance $SOURCE_INSTANCE..."
kubectl get pod "$SOURCE_INSTANCE" &>/dev/null || handle_error "Source instance $SOURCE_INSTANCE not found"

# Check if destination instance exists and is accessible
echo "Checking destination instance $DESTINATION_INSTANCE..."
kubectl get pod "$DESTINATION_INSTANCE" &>/dev/null || handle_error "Destination instance $DESTINATION_INSTANCE not found"

# Check if the database exists in the source instance
echo "Checking if database $DB_NAME exists in $SOURCE_INSTANCE..."
exec_psql "$SOURCE_INSTANCE" "\l" | grep -q "$DB_NAME" || handle_error "Database $DB_NAME not found in source instance"

# Create backup on source instance
echo "Creating backup of $DB_NAME from $SOURCE_INSTANCE..."
kubectl exec "$SOURCE_INSTANCE" --container postgres -- pg_dump -U "$DB_USER" -d "$DB_NAME" -f "/tmp/$BACKUP_FILE" || handle_error "Failed to create backup"

# Copy backup file to local machine
echo "Copying backup file to local machine..."
kubectl cp "$SOURCE_INSTANCE:/tmp/$BACKUP_FILE" "./$BACKUP_FILE" -c postgres || handle_error "Failed to copy backup to local machine"

# Check if the database exists in the destination instance, create if not
echo "Checking if database $DB_NAME exists in $DESTINATION_INSTANCE..."
if ! exec_psql "$DESTINATION_INSTANCE" "\l" | grep -q "$DB_NAME"; then
    echo "Creating database $DB_NAME in $DESTINATION_INSTANCE..."
    exec_psql "$DESTINATION_INSTANCE" "CREATE DATABASE $DB_NAME;" || handle_error "Failed to create database in destination instance"
fi

# Copy backup file to destination instance
echo "Copying backup file to $DESTINATION_INSTANCE..."
kubectl cp "./$BACKUP_FILE" "$DESTINATION_INSTANCE:/tmp/$BACKUP_FILE" -c postgres || handle_error "Failed to copy backup to destination instance"

# Restore from backup on destination instance
echo "Restoring backup on $DESTINATION_INSTANCE..."
kubectl exec "$DESTINATION_INSTANCE" --container postgres -- psql -U "$DB_USER" -d "$DB_NAME" -f "/tmp/$BACKUP_FILE" || handle_error "Failed to restore backup"

# Verify data restoration
echo "Verifying data restoration in $DESTINATION_INSTANCE..."
exec_psql "$DESTINATION_INSTANCE" "\dt" || handle_error "Failed to verify data restoration"

echo "Migration completed successfully!"
```
