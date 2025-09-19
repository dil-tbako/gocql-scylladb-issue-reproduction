# GoCQL ScyllaDB Driver Issue Reproduction

Reproduction case for DELETE operations failing on composite primary key tables.

## Problem
DELETE by partition key only fails silently on certain platforms with ScyllaDB gocql v1.14.0

## Quick Start
```bash
# Start ScyllaDB
docker-compose up -d

# Wait for ScyllaDB to be ready (may take 1-2 minutes)
# Note: On macOS, you may need to increase aio-max-nr:
# echo 'fs.aio-max-nr = 1048576' | sudo tee -a /etc/sysctl.conf

# Setup the schema
docker-compose exec scylladb cqlsh -f /scripts/setup-schema.cql

# Run tests
go test ./tests/... -v

# Manual verification
./scripts/manual-verification.sh
```

## Known Issues
- **macOS/Docker**: ScyllaDB may fail to start on macOS due to AIO limits. This reproduction case works best on Linux environments or CI systems.
- **Port Conflicts**: Default ports are mapped to 9043, 9161, 10001 to avoid conflicts with existing services.

## Platforms Tested
- ✅ Linux AMD64 (CI)
- ❌ macOS ARM64 (M3)
- ❌ Docker linux/amd64 on macOS M3

## Issue Details

### Problem Summary
- **Driver**: `github.com/scylladb/gocql v1.14.0` (ScyllaDB fork)
- **Issue**: `DELETE FROM table WHERE partition_key = ?` fails silently on macOS M3 but works on Linux AMD64
- **Table Structure**: Composite primary key `PRIMARY KEY (partition_key, clustering_key)`

### Reproduction Steps
1. Create table with composite primary key
2. Insert multiple rows with same partition key
3. Execute `DELETE FROM table WHERE partition_key = ?`
4. Verify deletion - on affected platforms, rows remain despite no error

### Expected vs Actual Behavior
- **Expected**: All rows with matching partition key are deleted
- **Linux AMD64**: ✅ Works correctly
- **macOS ARM64**: ❌ DELETE returns no error but rows remain
- **Docker linux/amd64 on macOS**: ❌ DELETE returns no error but rows remain

## Repository Structure
```
gocql-scylladb-issue-reproduction/
├── README.md
├── go.mod
├── go.sum
├── .gitignore
├── .github/workflows/test.yml
├── docker-compose.yml
├── scripts/
│   ├── setup-schema.cql
│   └── manual-verification.sh
├── pkg/client/scylla.go
└── tests/reproduction_test.go
```

## Repository
- **GitHub**: https://github.com/dil-tbako/gocql-scylladb-issue-reproduction
- **Clone**: `git clone https://github.com/dil-tbako/gocql-scylladb-issue-reproduction.git`

## Contributing
This is a bug reproduction repository. If you can reproduce or cannot reproduce the issue on your platform, please open an issue with your platform details.