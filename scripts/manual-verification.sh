#!/bin/bash

set -e

echo "=== Manual Verification Script ==="
echo "This script demonstrates that the CQL operations work correctly via cqlsh"
echo "but may fail via the gocql driver on certain platforms"
echo

# Wait for ScyllaDB to be ready
echo "Waiting for ScyllaDB to be ready..."
until docker-compose exec scylladb cqlsh -e "DESCRIBE KEYSPACES" > /dev/null 2>&1; do
    echo "ScyllaDB not ready yet, waiting..."
    sleep 5
done

echo "ScyllaDB is ready!"
echo

# Run manual CQL commands
echo "=== Running manual CQL verification ==="

# Setup
echo "1. Setting up test data..."
docker-compose exec scylladb cqlsh -e "
USE reproduction;

-- Clean up any existing data
DELETE FROM platform_users WHERE org_id = 'manual-test';

-- Insert test data
INSERT INTO platform_users (org_id, user_id, user_data, version, created_at)
VALUES ('manual-test', 'user-1', '{\"name\":\"Manual1\"}', 1, toTimestamp(now()));

INSERT INTO platform_users (org_id, user_id, user_data, version, created_at)
VALUES ('manual-test', 'user-2', '{\"name\":\"Manual2\"}', 1, toTimestamp(now()));

INSERT INTO platform_users (org_id, user_id, user_data, version, created_at)
VALUES ('manual-test', 'user-3', '{\"name\":\"Manual3\"}', 1, toTimestamp(now()));
"

# Verify insertion
echo "2. Verifying data insertion..."
COUNT_BEFORE=$(docker-compose exec scylladb cqlsh -e "USE reproduction; SELECT COUNT(*) FROM platform_users WHERE org_id = 'manual-test';" | grep -o '[0-9]\+' | tail -1)
echo "Users before deletion: $COUNT_BEFORE"

# Delete by partition key
echo "3. Deleting by partition key..."
docker-compose exec scylladb cqlsh -e "
USE reproduction;
DELETE FROM platform_users WHERE org_id = 'manual-test';
"

# Verify deletion
echo "4. Verifying deletion..."
COUNT_AFTER=$(docker-compose exec scylladb cqlsh -e "USE reproduction; SELECT COUNT(*) FROM platform_users WHERE org_id = 'manual-test';" | grep -o '[0-9]\+' | tail -1)
echo "Users after deletion: $COUNT_AFTER"

echo
if [ "$COUNT_AFTER" = "0" ]; then
    echo "✅ SUCCESS: Manual CQL deletion worked correctly"
    echo "   This proves the operation should work via gocql driver too"
else
    echo "❌ UNEXPECTED: Manual CQL deletion failed"
    echo "   This indicates a deeper ScyllaDB issue"
fi

echo
echo "=== Platform Information ==="
echo "Host OS: $(uname -s)"
echo "Host Architecture: $(uname -m)"
echo "Docker Version: $(docker --version)"

echo
echo "=== Next Steps ==="
echo "1. Run 'go test ./tests/... -v' to see if gocql driver behaves the same way"
echo "2. Compare results between manual verification and Go test"
echo "3. If they differ, this confirms the gocql driver issue"