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

### 2. Unnecessary Transactions for Read Operations (HIGH PRIORITY) ⏳ PENDING

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

**Impact**:
- Holds database connections longer than necessary
- Can exhaust connection pool under concurrent load
- Unnecessary overhead for read operations
- Can cause lock contention

**Solution**: Remove transaction wrapper for read-only operations. Use the pool directly:
```go
// Instead of s.tx(ctx), use s.q directly for reads
qSCIMUsers, err := s.q.ListSCIMUsers(ctx, ...)
```

### 3. No Connection Pool Configuration (MEDIUM PRIORITY) ⏳ PENDING

**Location**: `cmd/api/main.go:89` and `cmd/auth/main.go:64`

**Problem**: Connection pool is created with defaults:
```go
db, err := pgxpool.New(context.Background(), config.DB)
```

Default pgxpool settings:
- `MaxConns`: 25 (may be too low for production)
- `MinConns`: 0
- No connection lifetime limits
- No acquire timeout configuration

**Solution**: Configure connection pool explicitly:
```go
config, err := pgxpool.ParseConfig(dbURL)
if err != nil {
    panic(err)
}
config.MaxConns = 100  // Adjust based on your needs
config.MinConns = 5
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute

db, err := pgxpool.NewWithConfig(context.Background(), config)
```

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
