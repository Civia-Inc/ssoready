package testutil

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Civia-Inc/ssoready/internal/saml"
	"github.com/Civia-Inc/ssoready/internal/store"
	"github.com/Civia-Inc/ssoready/internal/store/idformat"
	"github.com/Civia-Inc/ssoready/internal/store/queries"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// getProjectRoot finds the project root by looking for go.mod
func getProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(fmt.Sprintf("%s/go.mod", dir)); err == nil {
			return dir
		}
		parent := fmt.Sprintf("%s/..", dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

// getStoreDB gets the database pool from store
func getStoreDB(s *store.Store) *pgxpool.Pool {
	return s.DB()
}

// TestAppOrganization represents a test app organization with its environment
type TestAppOrganization struct {
	ID          uuid.UUID
	Environment *TestEnvironment
}

// TestEnvironment represents a test environment
type TestEnvironment struct {
	ID               uuid.UUID
	AppOrgID         uuid.UUID
	RedirectURL      string
	OAuthRedirectURI *string
	AuthURL          *string
}

// TestOrganization represents a test organization within an environment
type TestOrganization struct {
	ID          uuid.UUID
	Environment *TestEnvironment
	ExternalID  *string
	DisplayName *string
	Domains     []string
}

// TestSAMLConnection represents a test SAML connection
type TestSAMLConnection struct {
	ID             uuid.UUID
	Organization   *TestOrganization
	IDPEntityID    string
	IDPRedirectURL string
	IDPCertificate []byte
	SPEntityID     string
	SPACSUrl       string
	IsPrimary      bool
}

// CreateTestAppOrganization creates a test app organization with an environment
func CreateTestAppOrganization(t *testing.T, s *store.Store, redirectURL string) *TestAppOrganization {
	t.Helper()
	ctx := context.Background()

	appOrgID := uuid.New()
	envID := uuid.New()
	oauthRedirectURI := "http://localhost:3000/callback"

	db := getStoreDB(s)

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Insert app organization
	_, err = tx.Exec(ctx, `INSERT INTO app_organizations (id) VALUES ($1) ON CONFLICT DO NOTHING`, appOrgID)
	require.NoError(t, err)

	// Create environment
	_, err = tx.Exec(ctx, `
		INSERT INTO environments (id, app_organization_id, redirect_url, oauth_redirect_uri, auth_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
	`, envID, appOrgID, redirectURL, oauthRedirectURI, "http://localhost:8081")
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	return &TestAppOrganization{
		ID: appOrgID,
		Environment: &TestEnvironment{
			ID:               envID,
			AppOrgID:         appOrgID,
			RedirectURL:      redirectURL,
			OAuthRedirectURI: &oauthRedirectURI,
			AuthURL:          stringPtr("http://localhost:8081"),
		},
	}
}

// CreateTestOrganization creates a test organization with domains
func CreateTestOrganization(t *testing.T, s *store.Store, env *TestEnvironment, externalID string, domains []string) *TestOrganization {
	t.Helper()
	ctx := context.Background()

	orgID := uuid.New()

	db := getStoreDB(s)
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := queries.New(tx)

	// Create organization
	_, err = q.CreateOrganization(ctx, queries.CreateOrganizationParams{
		ID:            orgID,
		EnvironmentID: env.ID,
		ExternalID:    &externalID,
		DisplayName:   stringPtr("Test Organization"),
	})
	require.NoError(t, err)

	// Create organization domains
	var orgDomains []string
	for _, domain := range domains {
		_, err = q.CreateOrganizationDomain(ctx, queries.CreateOrganizationDomainParams{
			ID:             uuid.New(),
			OrganizationID: orgID,
			Domain:         domain,
		})
		require.NoError(t, err)
		orgDomains = append(orgDomains, domain)
	}

	err = tx.Commit(ctx)
	require.NoError(t, err)

	return &TestOrganization{
		ID:          orgID,
		Environment: env,
		ExternalID:  &externalID,
		DisplayName: stringPtr("Test Organization"),
		Domains:     orgDomains,
	}
}

// CreateTestSAMLConnection creates a test SAML connection with IdP configuration
func CreateTestSAMLConnection(t *testing.T, store *store.Store, org *TestOrganization, idpEntityID, idpRedirectURL string, certificatePath string, isPrimary bool) *TestSAMLConnection {
	t.Helper()
	ctx := context.Background()

	samlConnID := uuid.New()
	authURL := "http://localhost:8081"
	if org.Environment.AuthURL != nil {
		authURL = *org.Environment.AuthURL
	}

	spEntityID := fmt.Sprintf("%s/v1/saml/%s", authURL, idformat.SAMLConnection.Format(samlConnID))
	spACSUrl := fmt.Sprintf("%s/v1/saml/%s/acs", authURL, idformat.SAMLConnection.Format(samlConnID))

	var idpCert []byte
	if certificatePath != "" {
		certData, err := os.ReadFile(certificatePath)
		require.NoError(t, err)

		block, _ := pem.Decode(certData)
		require.NotNil(t, block)
		_, err = x509.ParseCertificate(block.Bytes)
		require.NoError(t, err)

		idpCert = block.Bytes
	}

	db := getStoreDB(store)
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := queries.New(tx)

	_, err = q.CreateSAMLConnection(ctx, queries.CreateSAMLConnectionParams{
		ID:                 samlConnID,
		OrganizationID:     org.ID,
		IsPrimary:          isPrimary,
		SpAcsUrl:           spACSUrl,
		SpEntityID:         spEntityID,
		IdpEntityID:        &idpEntityID,
		IdpRedirectUrl:     &idpRedirectURL,
		IdpX509Certificate: idpCert,
	})
	require.NoError(t, err)

	if isPrimary {
		err = q.UpdatePrimarySAMLConnection(ctx, queries.UpdatePrimarySAMLConnectionParams{
			OrganizationID: org.ID,
			ID:             samlConnID,
		})
		require.NoError(t, err)
	}

	err = tx.Commit(ctx)
	require.NoError(t, err)

	return &TestSAMLConnection{
		ID:             samlConnID,
		Organization:   org,
		IDPEntityID:    idpEntityID,
		IDPRedirectURL: idpRedirectURL,
		IDPCertificate: idpCert,
		SPEntityID:     spEntityID,
		SPACSUrl:       spACSUrl,
		IsPrimary:      isPrimary,
	}
}

// CreateTestSAMLConnectionFromMetadata creates a SAML connection using metadata from testdata
func CreateTestSAMLConnectionFromMetadata(t *testing.T, store *store.Store, org *TestOrganization, provider string, isPrimary bool) *TestSAMLConnection {
	t.Helper()

	// Find the project root to locate testdata
	projectRoot := getProjectRoot()
	metadataPath := fmt.Sprintf("%s/internal/saml/testdata/assertions/%s/metadata.xml", projectRoot, provider)
	metadata, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	// Parse metadata to extract entity ID, redirect URL, and certificate
	metadataRes, err := saml.ParseMetadata(metadata)
	require.NoError(t, err)

	// Extract certificate bytes
	certBytes := metadataRes.IDPCertificate.Raw

	return CreateTestSAMLConnectionWithCertBytes(t, store, org, metadataRes.IDPEntityID, metadataRes.RedirectURL, certBytes, isPrimary)
}

// CreateTestSAMLConnectionWithCertBytes creates a SAML connection with certificate bytes
func CreateTestSAMLConnectionWithCertBytes(t *testing.T, store *store.Store, org *TestOrganization, idpEntityID, idpRedirectURL string, certBytes []byte, isPrimary bool) *TestSAMLConnection {
	t.Helper()
	ctx := context.Background()

	samlConnID := uuid.New()
	authURL := "http://localhost:8081"
	if org.Environment.AuthURL != nil {
		authURL = *org.Environment.AuthURL
	}

	spEntityID := fmt.Sprintf("%s/v1/saml/%s", authURL, idformat.SAMLConnection.Format(samlConnID))
	spACSUrl := fmt.Sprintf("%s/v1/saml/%s/acs", authURL, idformat.SAMLConnection.Format(samlConnID))

	db := getStoreDB(store)
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := queries.New(tx)

	_, err = q.CreateSAMLConnection(ctx, queries.CreateSAMLConnectionParams{
		ID:                 samlConnID,
		OrganizationID:     org.ID,
		IsPrimary:          isPrimary,
		SpAcsUrl:           spACSUrl,
		SpEntityID:         spEntityID,
		IdpEntityID:        &idpEntityID,
		IdpRedirectUrl:     &idpRedirectURL,
		IdpX509Certificate: certBytes,
	})
	require.NoError(t, err)

	if isPrimary {
		err = q.UpdatePrimarySAMLConnection(ctx, queries.UpdatePrimarySAMLConnectionParams{
			OrganizationID: org.ID,
			ID:             samlConnID,
		})
		require.NoError(t, err)
	}

	err = tx.Commit(ctx)
	require.NoError(t, err)

	return &TestSAMLConnection{
		ID:             samlConnID,
		Organization:   org,
		IDPEntityID:    idpEntityID,
		IDPRedirectURL: idpRedirectURL,
		IDPCertificate: certBytes,
		SPEntityID:     spEntityID,
		SPACSUrl:       spACSUrl,
		IsPrimary:      isPrimary,
	}
}

// TestAPIKey represents a test API key
type TestAPIKey struct {
	ID               uuid.UUID
	SecretToken      string
	EnvironmentID    uuid.UUID
	AppOrgID         uuid.UUID
	HasManagementAPI bool
}

// CreateTestAPIKey creates a test API key for an environment
func CreateTestAPIKey(t *testing.T, s *store.Store, env *TestEnvironment, hasManagementAPI bool) *TestAPIKey {
	t.Helper()
	ctx := context.Background()

	db := getStoreDB(s)
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := queries.New(tx)

	secretValue := uuid.New()
	secretValueSHA := sha256.Sum256(secretValue[:])

	apiKeyID := uuid.New()
	_, err = q.CreateAPIKey(ctx, queries.CreateAPIKeyParams{
		ID:                     apiKeyID,
		SecretValueSha256:      secretValueSHA[:],
		EnvironmentID:          env.ID,
		HasManagementApiAccess: &hasManagementAPI,
	})
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	return &TestAPIKey{
		ID:               apiKeyID,
		SecretToken:      idformat.APISecretKey.Format(secretValue),
		EnvironmentID:    env.ID,
		AppOrgID:         env.AppOrgID,
		HasManagementAPI: hasManagementAPI,
	}
}

// TestSAMLAccessCode represents a SAML access code for testing
type TestSAMLAccessCode struct {
	Code        string // The formatted access code (e.g., "saml_access_code_...")
	SAMLFlowID  uuid.UUID
	Email       string
	State       string
	Organization *TestOrganization
}

// CreateTestSAMLAccessCode creates a SAML flow with an access code for testing
// This simulates a successful SAML assertion being processed
func CreateTestSAMLAccessCode(t *testing.T, s *store.Store, samlConn *TestSAMLConnection, email string, state string) *TestSAMLAccessCode {
	t.Helper()
	ctx := context.Background()

	// Generate access code UUID
	accessCodeUUID := uuid.New()
	accessCodeSHA := sha256.Sum256(accessCodeUUID[:])

	samlFlowID := uuid.New()
	now := time.Now()
	attrs := map[string]string{"email": email}
	attrsJSON, err := json.Marshal(attrs)
	require.NoError(t, err)

	db := getStoreDB(s)
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	q := queries.New(tx)

	// Create SAML flow with access code (simulating successful assertion processing)
	_, err = q.UpsertSAMLFlowReceiveAssertion(ctx, queries.UpsertSAMLFlowReceiveAssertionParams{
		ID:                   samlFlowID,
		SamlConnectionID:     samlConn.ID,
		AccessCodeSha256:     accessCodeSHA[:],
		ExpireTime:           time.Now().Add(time.Hour),
		State:                state,
		CreateTime:           now,
		UpdateTime:           now,
		Assertion:            stringPtr("<saml:Assertion>test</saml:Assertion>"),
		ReceiveAssertionTime: &now,
		Status:               queries.SamlFlowStatusInProgress,
	})
	require.NoError(t, err)

	// Update subject data (email and attributes)
	_, err = q.UpdateSAMLFlowSubjectData(ctx, queries.UpdateSAMLFlowSubjectDataParams{
		ID:                   samlFlowID,
		Email:                &email,
		SubjectIdpAttributes: attrsJSON,
	})
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	return &TestSAMLAccessCode{
		Code:         idformat.SAMLAccessCode.Format(accessCodeUUID),
		SAMLFlowID:   samlFlowID,
		Email:        email,
		State:        state,
		Organization: samlConn.Organization,
	}
}

// stringPtr returns a pointer to the string
func stringPtr(s string) *string {
	return &s
}
