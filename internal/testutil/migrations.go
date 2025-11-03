package testutil

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// RunMigrations runs database migrations using the migrate command
func RunMigrations(t *testing.T, dbURL string) {
	t.Helper()

	ctx := context.Background()

	// Check if migrate binary exists, otherwise use go run
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/migrate", "--database", dbURL, "up")
	// Note: getProjectRoot() is defined in fixtures.go
	cmd.Dir = getProjectRoot()

	// Capture stderr to check for "no change" message
	var stderr bytes.Buffer
	cmd.Stdout = os.Stderr // Still show output
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// golang-migrate returns exit code 1 with "no change" message when migrations
		// are already up to date - this is actually a success case
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "no change") {
			// Migrations are already up to date, which is fine
			return
		}
		// Real error occurred
		require.NoError(t, err, "Failed to run migrations: %s", stderrStr)
	}
}
