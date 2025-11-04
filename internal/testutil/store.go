package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/Civia-Inc/ssoready/internal/hexkey"
	"github.com/Civia-Inc/ssoready/internal/pagetoken"
	"github.com/Civia-Inc/ssoready/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// SetupTestStore creates a store instance for testing using the DATABASE_URL environment variable
// If DATABASE_URL is not set, tests will be skipped
// This function automatically runs migrations before returning the store
func SetupTestStore(t *testing.T) *store.Store {
	t.Helper()

	dbURL := getTestDBURL()
	if dbURL == "" {
		t.Skip("DATABASE_URL not set - skipping integration test")
	}

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
