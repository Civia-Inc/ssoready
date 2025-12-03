# Intermittent API Timeout Issues - Analysis & Recommendations

## Problem
Intermittent timeouts (60+ seconds) on SCIM users API endpoint: `/v1/scim/users?organizationId=...&pageToken=...`

## Root Causes Identified

### 1. Missing Database Indexes (HIGH PRIORITY) ✅ COMPLETED

**Status**: Migration `000054_scim_users_indexes` has been created and is ready to be applied.

The pagination queries are missing critical indexes:

#### Issue: `ListSCIMUsers` query
```sql
SELECT * FROM scim_users
WHERE scim_directory_id = $1 AND id >= $2
ORDER BY id
LIMIT $3;
```

**Problem**: No composite index on `(scim_directory_id, id)`. With large datasets, this requires a full table scan or inefficient index usage.

**Solution**: ✅ Added index in migration `000054_scim_users_indexes.up.sql`:
```sql
CREATE INDEX IF NOT EXISTS scim_users_scim_directory_id_id
    ON scim_users(scim_directory_id, id);
```

#### Issue: `ListSCIMUsersInSCIMGroup` query
```sql
SELECT * FROM scim_users
WHERE scim_users.scim_directory_id = $1
  AND scim_users.id >= $2
  AND EXISTS(
    SELECT * FROM scim_user_group_memberships
    WHERE scim_group_id = $4
      AND scim_user_id = scim_users.id
  )
ORDER BY scim_users.id
LIMIT $3;
```

**Problem**: The EXISTS subquery may be slow without proper indexes on `scim_user_group_memberships`.

**Solution**: ✅ Added index in migration `000054_scim_users_indexes.up.sql`:
```sql
CREATE INDEX IF NOT EXISTS scim_user_group_memberships_scim_group_id_scim_user_id
    ON scim_user_group_memberships(scim_group_id, scim_user_id);
```

Note: The existing unique constraint `scim_user_group_memberships_scim_user_id_scim_group_id_key` provides an index on `(scim_user_id, scim_group_id)`, but the new index with reversed column order is more optimal for queries filtering by `scim_group_id` first.

### 2. Unnecessary Transactions for Read Operations (MEDIUM-HIGH PRIORITY) ⏳ PENDING

**Location**: `internal/store/scim.go:17`

**Problem**: The `ListSCIMUsers` function uses a transaction for read-only operations:
```go
_, q, commit, rollback, err := s.tx(ctx)
defer rollback()
// ... read-only queries ...
if err := commit(); err != nil {
    return nil, fmt.Errorf("commit: %w", err)
}
```

**Analysis - Does the transaction provide consistency?**

PostgreSQL transactions with default `READ COMMITTED` isolation level provide snapshot isolation - all queries in the transaction see a consistent snapshot of the database. However, **the transaction is NOT actually providing consistency** because the code mixes queries inside and outside the transaction:

- Line 41: Uses `s.q` (outside transaction) - `GetSCIMDirectoryByIDAndEnvironmentID`
- Lines 53/61: Use `q` (inside transaction) - `GetPrimarySCIMDirectoryIDByOrganizationID`
- Lines 89/100: Use `s.q` (outside transaction) - `ListSCIMUsers` / `ListSCIMUsersInSCIMGroup`

Since the main data query (lines 89/100) uses `s.q` outside the transaction, it sees potentially different data than the lookup queries inside the transaction. The transaction is not providing the intended consistency benefit.

**Additional Notes**:
- PostgreSQL uses MVCC (Multi-Version Concurrency Control) - read transactions don't lock tables
- The transaction does NOT prevent other transactions from modifying data
- The transaction only provides snapshot isolation for queries that use the transaction query object `q`
- **Read transactions do NOT cause lock contention** - they don't block other read or write operations

**Impact**:
- **Primary Issue**: Holds database connections longer than necessary, which can exhaust the connection pool under concurrent load
  - When the pool is exhausted, new requests must wait for a connection to become available
  - This waiting can contribute to timeouts, especially under high load
- Unnecessary overhead for read operations
- Does NOT actually provide consistency (due to mixed `s.q`/`q` usage)
- Does NOT cause lock contention (read transactions don't block other operations)

**Priority Assessment for Timeout Reduction**:
- **Medium-High Priority** (not critical, but still important)
- The connection pool exhaustion is a real issue that can contribute to timeouts
- However, it's less urgent than if there were actual lock contention blocking operations
- The database indexes (already completed) will likely have a bigger impact on timeout reduction
- This should be addressed, but may not be the primary cause of the 60-second timeouts

**Solution**: Remove transaction wrapper for read-only operations. Use the pool directly:
```go
// Instead of s.tx(ctx), use s.q directly for reads
qSCIMUsers, err := s.q.ListSCIMUsers(ctx, ...)
```

**Note**: If consistency across multiple queries is truly needed, all queries must use the transaction query object `q`, not `s.q`. However, for this read-only pagination use case, consistency across queries is typically not required.

### 3. No Connection Pool Configuration (MEDIUM PRIORITY) ✅ COMPLETED

**Status**: Connection pool configuration has been implemented in both `cmd/api/main.go` and `cmd/auth/main.go`.

**Location**: `cmd/api/main.go:90-100` and `cmd/auth/main.go:65-75`

**Problem**: Connection pool was created with defaults:
```go
db, err := pgxpool.New(context.Background(), config.DB)
```

Default pgxpool settings:
- `MaxConns`: 25 (may be too low for production)
- `MinConns`: 0
- No connection lifetime limits
- No acquire timeout configuration

**Solution**: ✅ Configured connection pool explicitly in both services.


### 4. No Query Timeouts (MEDIUM PRIORITY) ⏳ PENDING

**Problem**: Database queries don't have explicit timeouts. If a query gets stuck, it can wait indefinitely.

**Solution**: Add context timeouts for database operations:
```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

qSCIMUsers, err := s.q.ListSCIMUsers(ctx, ...)
```

Or configure at the pool level:
```go
config.ConnConfig.ConnectTimeout = 5 * time.Second
```

### 5. Mixed Transaction Usage (LOW PRIORITY) ⏳ PENDING

**Location**: `internal/store/scim.go:41`

**Problem**: Line 41 uses `s.q` (outside transaction) while other queries use `q` (inside transaction):
```go
if _, err := s.q.GetSCIMDirectoryByIDAndEnvironmentID(ctx, ...); err != nil {
```

This adds an extra database round-trip and breaks transaction consistency.

**Solution**: Use the transaction query object `q` consistently:
```go
if _, err := q.GetSCIMDirectoryByIDAndEnvironmentID(ctx, ...); err != nil {
```
