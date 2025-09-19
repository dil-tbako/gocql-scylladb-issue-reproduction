# GitHub Issue Template

## DELETE by partition key fails silently on certain platforms - ScyllaDB gocql v1.14.0

**Summary**  
DELETE operations targeting composite primary key tables by partition key only fail silently on macOS ARM64 and Docker linux/amd64 running on macOS ARM64, while working correctly on native Linux AMD64 systems.

**Environment**
- **Driver**: `github.com/scylladb/gocql v1.14.0`
- **Database**: ScyllaDB 5.2 (Docker)
- **Failing Platforms**: 
  - macOS ARM64 (M3)  
  - Docker linux/amd64 on macOS ARM64
- **Working Platforms**:
  - Linux AMD64 (CI environments)

**Reproduction**  
Complete reproduction repository available at: https://github.com/tbako/gocql-scylladb-issue-reproduction

### Quick Reproduction
```bash
git clone https://github.com/tbako/gocql-scylladb-issue-reproduction
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

**Expected Behavior**  
DELETE by partition key should remove all rows with that partition key value.

**Actual Behavior**
- **Linux AMD64**: Works correctly - all rows deleted
- **macOS ARM64**: Fails silently - no error returned but rows remain
- **Docker linux/amd64 on macOS ARM64**: Fails silently - no error returned but rows remain

**Manual Verification**  
The same CQL operation works correctly when executed via `cqlsh`:
```sql
-- This works in cqlsh on all platforms
DELETE FROM platform_users WHERE org_id = 'test-org';
```

**Additional Context**  
This appears to be specific to the ScyllaDB fork of gocql. The issue affects production workloads that rely on partition-level deletions for data cleanup operations.