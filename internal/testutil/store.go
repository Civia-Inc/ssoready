package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Civia-Inc/ssoready/internal/hexkey"
	"github.com/Civia-Inc/ssoready/internal/pagetoken"
	"github.com/Civia-Inc/ssoready/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// SetupTestStore creates a store instance for testing using the DATABASE_URL environment variable
// If DATABASE_URL is not set, tests will be skipped
// This function automatically creates the test database if it doesn't exist and runs migrations
func SetupTestStore(t *testing.T) *store.Store {
	t.Helper()

	dbURL := getTestDBURL()
	if dbURL == "" {
		t.Skip("DATABASE_URL not set - skipping integration test")
	}

	// Ensure the test database exists
	ensureTestDatabase(t, dbURL)

	// Run migrations first
	RunMigrations(t, dbURL)

	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	// Create test signing keys
	pageEncodingValue := [32]byte{}
	_, err = rand.Read(pageEncodingValue[:])
	require.NoError(t, err)

	samlStateSigningKey := [32]byte{}
	_, err = rand.Read(samlStateSigningKey[:])
	require.NoError(t, err)

	pageEncodingSecret, err := hexkey.New(hex.EncodeToString(pageEncodingValue[:]))
	require.NoError(t, err)

	s := store.New(store.NewStoreParams{
		DB:                       pool,
		PageEncoder:              pagetoken.Encoder{Secret: pageEncodingSecret},
		DefaultAuthURL:           "http://localhost:8081",
		DefaultAdminSetupURL:     "http://localhost:8082",
		DefaultAdminTestModeURL:  "http://localhost:8082/test",
		SAMLStateSigningKey:      samlStateSigningKey,
		DisableNewAppOrgCreation: false, // Tests should allow org creation by default
	})

	return s
}

func getTestDBURL() string {
	// Check for standard test database URL
	// If not set, return empty string to skip tests (especially important for CI)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		return dbURL
	}

	// No fallback - tests requiring a database will be skipped
	// For local development, explicitly set DATABASE_URL:
	//   export DATABASE_URL="postgres://postgres:password@localhost:5433/ssoready_test?sslmode=disable"
	return ""
}

// ensureTestDatabase creates the test database if it doesn't exist
func ensureTestDatabase(t *testing.T, dbURL string) {
	t.Helper()

	// Parse the database URL to extract components
	parsedURL, err := url.Parse(dbURL)
	require.NoError(t, err, "Failed to parse DATABASE_URL")

	// Extract database name from path (remove leading slash)
	dbName := strings.TrimPrefix(parsedURL.Path, "/")
	if dbName == "" {
		// If no database name specified, use "postgres" as default
		// "postgres" always exists, so we don't need to create it
		return
	}

	// Don't try to create the "postgres" database - it always exists
	if dbName == "postgres" {
		return
	}

	// Create a connection URL to the "postgres" database (which always exists)
	// to check if our test database exists
	adminURL := *parsedURL
	adminURL.Path = "/postgres"

	// Connect to postgres database to check/create test database
	adminPool, err := pgxpool.New(context.Background(), adminURL.String())
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	defer adminPool.Close()

	// Check if database exists
	ctx := context.Background()
	var exists bool
	err = adminPool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	require.NoError(t, err, "Failed to check if database exists")

	if !exists {
		// Create the database
		// Note: We can't use parameterized queries for CREATE DATABASE
		// so we need to be careful about SQL injection, but since dbName comes from
		// the URL path, it should be safe. We'll validate it contains only safe characters.
		if !isSafeDatabaseName(dbName) {
			require.Fail(t, "Database name contains unsafe characters: %s", dbName)
		}

		// Terminate any existing connections to the database before creating it
		// (in case it's being dropped/recreated)
		_, err = adminPool.Exec(ctx, "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()", dbName)
		// Ignore errors here - database might not exist yet

		// Create the database with proper quoting
		// Use pgx.Identifier to safely quote the database name
		quotedName := `"` + strings.ReplaceAll(dbName, `"`, `""`) + `"`
		_, err = adminPool.Exec(ctx, "CREATE DATABASE "+quotedName)
		require.NoError(t, err, "Failed to create test database: %s", dbName)
	}
}

// isSafeDatabaseName checks if a database name contains only safe characters
// PostgreSQL identifiers can contain letters, digits, underscores, and dollar signs
func isSafeDatabaseName(name string) bool {
	if name == "" {
		return false
	}
	// Check that it only contains alphanumeric, underscore, and dollar sign
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '$') {
			return false
		}
	}
	return true
}
