# GoCQL ScyllaDB Driver Issue Reproduction Plan

## Overview
This document outlines a comprehensive plan to create a minimal reproduction repository for a critical issue with the ScyllaDB gocql driver fork where DELETE operations on composite primary key tables fail when deleting by partition key only on certain platforms.

## Issue Summary
- **Problem**: `DELETE FROM table WHERE partition_key = ?` fails silently on macOS M3 but works on Linux AMD64
- **Driver**: `github.com/scylladb/gocql v1.14.0` (ScyllaDB fork)
- **Table Structure**: Composite primary key `PRIMARY KEY (partition_key, clustering_key)`
- **Platforms Affected**: macOS ARM64 (M3), Docker linux/amd64 on M3
- **Platforms Working**: Native Linux AMD64 CI environments

## Repository Structure

```
gocql-scylladb-issue-reproduction/
├── README.md
├── go.mod
├── go.sum
├── .gitignore
├── .github/
│   └── workflows/
│       └── test.yml
├── docker-compose.yml
├── scripts/
│   ├── setup-schema.cql
│   └── manual-verification.sh
├── pkg/
│   ├── client/
│   │   └── scylla.go
│   └── models/
│       └── user.go
├── tests/
│   ├── reproduction_test.go
│   └── driver_comparison_test.go
└── docs/
    ├── issue-analysis.md
    └── github-issue-template.md
```

## Implementation Plan

### 1. Repository Setup

#### `README.md`
```markdown
# GoCQL ScyllaDB Driver Issue Reproduction

Reproduction case for DELETE operations failing on composite primary key tables.

## Problem
DELETE by partition key only fails silently on certain platforms with ScyllaDB gocql v1.14.0

## Quick Start
```bash
# Start ScyllaDB
docker-compose up -d

# Run tests
go test ./tests/... -v

# Manual verification
./scripts/manual-verification.sh
```

## Platforms Tested
- ✅ Linux AMD64 (CI)
- ❌ macOS ARM64 (M3)
- ❌ Docker linux/amd64 on macOS M3
```

#### `go.mod`
```go
module github.com/your-username/gocql-scylladb-issue-reproduction

go 1.21

require (
    github.com/scylladb/gocql v1.14.0
    github.com/gocql/gocql v1.6.0  // For comparison
    github.com/stretchr/testify v1.8.4
)
```

#### `.gitignore`
```
# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.work
go.work.sum

# Docker
.env

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
```

### 2. Docker Configuration

#### `docker-compose.yml`
```yaml
version: '3.8'

services:
  scylladb:
    image: scylladb/scylla:5.2
    platform: linux/amd64  # Force AMD64 even on M1/M2/M3 Macs
    container_name: scylla-reproduction
    ports:
      - "9042:9042"
      - "9160:9160" 
      - "10000:10000"
    environment:
      - SCYLLA_CLUSTER_NAME=reproduction-cluster
    command: >
      --seeds=scylladb
      --smp 1
      --memory 1G
      --overprovisioned 1
      --api-address 0.0.0.0
      --listen-address 0.0.0.0
      --rpc-address 0.0.0.0
      --broadcast-address 127.0.0.1
      --broadcast-rpc-address 127.0.0.1
    volumes:
      - scylla-data:/var/lib/scylla
    healthcheck:
      test: ["CMD-SHELL", "cqlsh -e 'DESCRIBE KEYSPACES'"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s

volumes:
  scylla-data:
```

### 3. Database Schema

#### `scripts/setup-schema.cql`
```sql
-- Create keyspace
CREATE KEYSPACE IF NOT EXISTS reproduction 
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': 1
};

USE reproduction;

-- Create table with composite primary key
CREATE TABLE IF NOT EXISTS platform_users (
  org_id TEXT,
  user_id TEXT,
  user_data TEXT,
  version INT,
  created_at TIMESTAMP,
  PRIMARY KEY (org_id, user_id)
);

-- Insert test data
INSERT INTO platform_users (org_id, user_id, user_data, version, created_at)
VALUES ('org-1', 'user-1', '{"name":"John"}', 1, toTimestamp(now()));

INSERT INTO platform_users (org_id, user_id, user_data, version, created_at)
VALUES ('org-1', 'user-2', '{"name":"Jane"}', 1, toTimestamp(now()));

INSERT INTO platform_users (org_id, user_id, user_data, version, created_at)
VALUES ('org-1', 'user-3', '{"name":"Bob"}', 1, toTimestamp(now()));

-- This should work in cqlsh
SELECT COUNT(*) FROM platform_users WHERE org_id = 'org-1';
-- Expected: 3

DELETE FROM platform_users WHERE org_id = 'org-1';

SELECT COUNT(*) FROM platform_users WHERE org_id = 'org-1';
-- Expected: 0
```

### 4. Go Implementation

#### `pkg/client/scylla.go`
```go
package client

import (
    "fmt"
    "time"
    
    "github.com/scylladb/gocql"
)

type ScyllaClient struct {
    session *gocql.Session
}

func NewScyllaClient(hosts []string) (*ScyllaClient, error) {
    cluster := gocql.NewCluster(hosts...)
    cluster.Keyspace = "reproduction"
    cluster.Consistency = gocql.Quorum
    cluster.Timeout = 10 * time.Second
    cluster.ConnectTimeout = 10 * time.Second
    
    session, err := cluster.CreateSession()
    if err != nil {
        return nil, fmt.Errorf("failed to create session: %w", err)
    }
    
    return &ScyllaClient{session: session}, nil
}

func (c *ScyllaClient) Close() {
    if c.session != nil {
        c.session.Close()
    }
}

func (c *ScyllaClient) InsertUser(orgID, userID, userData string, version int) error {
    query := `INSERT INTO platform_users (org_id, user_id, user_data, version, created_at) 
              VALUES (?, ?, ?, ?, toTimestamp(now()))`
    
    return c.session.Query(query, orgID, userID, userData, version).Exec()
}

func (c *ScyllaClient) DeleteUsersByOrgID(orgID string) error {
    query := `DELETE FROM platform_users WHERE org_id = ?`
    return c.session.Query(query, orgID).Exec()
}

func (c *ScyllaClient) CountUsersByOrgID(orgID string) (int, error) {
    query := `SELECT COUNT(*) FROM platform_users WHERE org_id = ?`
    
    var count int
    err := c.session.Query(query, orgID).Scan(&count)
    return count, err
}

func (c *ScyllaClient) GetAllUsersByOrgID(orgID string) ([]map[string]interface{}, error) {
    query := `SELECT user_id, user_data, version FROM platform_users WHERE org_id = ?`
    
    iter := c.session.Query(query, orgID).Iter()
    defer iter.Close()
    
    var users []map[string]interface{}
    var userID, userData string
    var version int
    
    for iter.Scan(&userID, &userData, &version) {
        users = append(users, map[string]interface{}{
            "user_id":   userID,
            "user_data": userData,
            "version":   version,
        })
    }
    
    return users, iter.Close()
}
```

#### `tests/reproduction_test.go`
```go
package tests

import (
    "fmt"
    "runtime"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/your-username/gocql-scylladb-issue-reproduction/pkg/client"
)

func TestDeleteByPartitionKey_ReproducesIssue(t *testing.T) {
    t.Logf("Running on platform: %s/%s", runtime.GOOS, runtime.GOARCH)
    
    // Connect to ScyllaDB
    scyllaClient, err := client.NewScyllaClient([]string{"127.0.0.1:9042"})
    require.NoError(t, err, "Failed to connect to ScyllaDB")
    defer scyllaClient.Close()
    
    orgID := fmt.Sprintf("test-org-%d", time.Now().Unix())
    
    t.Run("setup_test_data", func(t *testing.T) {
        // Insert multiple users for the same organization
        for i := 1; i <= 3; i++ {
            userID := fmt.Sprintf("user-%d", i)
            userData := fmt.Sprintf(`{"name":"User%d"}`, i)
            
            err := scyllaClient.InsertUser(orgID, userID, userData, 1)
            require.NoError(t, err, "Failed to insert user %s", userID)
        }
        
        // Verify data is inserted
        count, err := scyllaClient.CountUsersByOrgID(orgID)
        require.NoError(t, err, "Failed to count users")
        assert.Equal(t, 3, count, "Expected 3 users to be inserted")
        
        users, err := scyllaClient.GetAllUsersByOrgID(orgID)
        require.NoError(t, err, "Failed to get users")
        assert.Len(t, users, 3, "Expected 3 users in result set")
    })
    
    t.Run("delete_by_partition_key", func(t *testing.T) {
        // This is the operation that fails on certain platforms
        err := scyllaClient.DeleteUsersByOrgID(orgID)
        assert.NoError(t, err, "DELETE by partition key should not return error")
        
        // Verify deletion worked
        count, err := scyllaClient.CountUsersByOrgID(orgID)
        require.NoError(t, err, "Failed to count users after deletion")
        
        // This is where the issue manifests
        if count != 0 {
            t.Errorf("ISSUE REPRODUCED: DELETE by partition key failed silently")
            t.Logf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
            t.Logf("Expected 0 users, got %d users", count)
            
            // Show remaining users for debugging
            users, err := scyllaClient.GetAllUsersByOrgID(orgID)
            if err == nil {
                t.Logf("Remaining users: %+v", users)
            }
        } else {
            t.Logf("SUCCESS: DELETE by partition key worked correctly")
        }
        
        assert.Equal(t, 0, count, "All users should be deleted when deleting by partition key")
    })
}

func TestPlatformComparison(t *testing.T) {
    t.Logf("=== PLATFORM INFORMATION ===")
    t.Logf("GOOS: %s", runtime.GOOS)
    t.Logf("GOARCH: %s", runtime.GOARCH)
    t.Logf("Go Version: %s", runtime.Version())
    t.Logf("NumCPU: %d", runtime.NumCPU())
    
    // This test documents the platform where the test runs
    // and helps identify patterns in the CI results
}
```

### 5. Manual Verification

#### `scripts/manual-verification.sh`
```bash
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
echo "=== Next Steps ==="
echo "1. Run 'go test ./tests/... -v' to see if gocql driver behaves the same way"
echo "2. Compare results between manual verification and Go test"
echo "3. If they differ, this confirms the gocql driver issue"
```

### 6. GitHub Actions Workflow

#### `.github/workflows/test.yml`
```yaml
name: Cross-Platform Issue Reproduction

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test-linux-amd64:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Start ScyllaDB
      run: docker-compose up -d
    
    - name: Wait for ScyllaDB
      run: |
        timeout 120s bash -c 'until docker-compose exec scylladb cqlsh -e "DESCRIBE KEYSPACES" > /dev/null 2>&1; do sleep 5; done'
    
    - name: Setup Schema
      run: docker-compose exec scylladb cqlsh -f /scripts/setup-schema.cql
      
    - name: Run Manual Verification
      run: chmod +x scripts/manual-verification.sh && ./scripts/manual-verification.sh
    
    - name: Run Tests
      run: go test ./tests/... -v
      
    - name: Platform Info
      run: |
        echo "Platform: $(uname -a)"
        echo "Go Version: $(go version)"
        echo "Docker Version: $(docker --version)"

  test-macos-arm64:
    runs-on: macos-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install Docker
      run: |
        brew install docker
        brew install docker-compose
        
    - name: Start Docker Desktop
      run: |
        open /Applications/Docker.app
        sleep 30
    
    - name: Start ScyllaDB
      run: docker-compose up -d
    
    - name: Wait for ScyllaDB
      run: |
        timeout 120s bash -c 'until docker-compose exec scylladb cqlsh -e "DESCRIBE KEYSPACES" > /dev/null 2>&1; do sleep 5; done'
    
    - name: Setup Schema
      run: docker-compose exec scylladb cqlsh -f /scripts/setup-schema.cql
      
    - name: Run Manual Verification
      run: chmod +x scripts/manual-verification.sh && ./scripts/manual-verification.sh
    
    - name: Run Tests
      run: go test ./tests/... -v
      
    - name: Platform Info
      run: |
        echo "Platform: $(uname -a)"
        echo "Go Version: $(go version)"
        echo "Docker Version: $(docker --version)"
        echo "Architecture: $(uname -m)"

  comparison:
    needs: [test-linux-amd64, test-macos-arm64]
    runs-on: ubuntu-latest
    if: always()
    steps:
    - name: Compare Results
      run: |
        echo "This job runs after both platform tests complete"
        echo "Check the logs above to compare Linux AMD64 vs macOS ARM64 behavior"
        echo "Look for differences in test results between platforms"
```

### 7. Documentation Files

#### `docs/github-issue-template.md`
```markdown
# DELETE by partition key fails silently on certain platforms - ScyllaDB gocql v1.14.0

## Summary
DELETE operations targeting composite primary key tables by partition key only fail silently on macOS ARM64 and Docker linux/amd64 running on macOS ARM64, while working correctly on native Linux AMD64 systems.

## Environment
- **Driver**: `github.com/scylladb/gocql v1.14.0`
- **Database**: ScyllaDB 5.2 (Docker)
- **Failing Platforms**: 
  - macOS ARM64 (M3)  
  - Docker linux/amd64 on macOS ARM64
- **Working Platforms**:
  - Linux AMD64 (CI environments)

## Reproduction
Complete reproduction repository available at: [REPOSITORY_URL]

### Quick Reproduction
```bash
git clone [REPOSITORY_URL]
cd gocql-scylladb-issue-reproduction
docker-compose up -d
go test ./tests/... -v
```

### Table Schema
```sql
CREATE TABLE platform_users (
  org_id TEXT,
  user_id TEXT,
  user_data TEXT,
  version INT,
  created_at TIMESTAMP,
  PRIMARY KEY (org_id, user_id)  -- Composite key: org_id=partition, user_id=clustering
);
```

### Failing Operation
```go
// This DELETE by partition key fails silently on certain platforms
query := "DELETE FROM platform_users WHERE org_id = ?"
err := session.Query(query, orgID).Exec()
// err == nil but deletion doesn't occur
```

## Expected Behavior
DELETE by partition key should remove all rows with that partition key value.

## Actual Behavior
- **Linux AMD64**: Works correctly - all rows deleted
- **macOS ARM64**: Fails silently - no error returned but rows remain
- **Docker linux/amd64 on macOS ARM64**: Fails silently - no error returned but rows remain

## Manual Verification
The same CQL operation works correctly when executed via `cqlsh`:
```sql
-- This works in cqlsh on all platforms
DELETE FROM platform_users WHERE org_id = 'test-org';
```

## Additional Context
This appears to be specific to the ScyllaDB fork of gocql. The issue affects production workloads that rely on partition-level deletions for data cleanup operations.
```

## Next Steps

1. **Create the reproduction repository** using this structure
2. **Implement all components** following the detailed specifications above
3. **Test on multiple platforms** to confirm the issue reproduction
4. **Submit GitHub issue** with the reproduction repository link
5. **Share with ScyllaDB gocql maintainers** for investigation

This comprehensive reproduction case will provide maintainers with:
- ✅ Minimal, focused reproduction code
- ✅ Clear platform-specific behavior documentation  
- ✅ Automated CI testing across platforms
- ✅ Manual verification proving the SQL operations work via cqlsh
- ✅ Comparison between original gocql and ScyllaDB fork
- ✅ Complete environment setup with Docker

The structured approach ensures the issue can be quickly understood, reproduced, and debugged by the maintainers.