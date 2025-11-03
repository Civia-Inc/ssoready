package apiservice

import (
	"context"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"connectrpc.com/connect"
	ssoreadyv1 "github.com/Civia-Inc/ssoready/internal/gen/ssoready/v1"
	"github.com/Civia-Inc/ssoready/internal/testutil"
	"github.com/stretchr/testify/assert"
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

func TestGetSAMLRedirectURL_HappyPath(t *testing.T) {
	store := testutil.SetupTestStore(t)
	ctx := context.Background()

	// Create test data
	appOrg := testutil.CreateTestAppOrganization(t, store, "http://localhost:3000/callback")
	org := testutil.CreateTestOrganization(t, store, appOrg.Environment, "test-org-123", []string{"example.com"})
	_ = testutil.CreateTestSAMLConnectionFromMetadata(t, store, org, "okta", true)
	apiKey := testutil.CreateTestAPIKey(t, store, appOrg.Environment, false)

	service := &Service{Store: store}
	ctx = testutil.WithAPIKeyContext(ctx, apiKey)

	req := connect.NewRequest(&ssoreadyv1.GetSAMLRedirectURLRequest{
		OrganizationExternalId: "test-org-123",
	})

	resp, err := service.GetSAMLRedirectURL(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.RedirectUrl)
	// The redirect URL points to the init endpoint, not the final SAML request
	// The SAMLRequest is generated when the init endpoint is called
	assert.Contains(t, resp.Msg.RedirectUrl, "/v1/saml/")
	assert.Contains(t, resp.Msg.RedirectUrl, "/init")
	assert.Contains(t, resp.Msg.RedirectUrl, "state=")
}

func TestGetSAMLRedirectURL_InvalidOrganization(t *testing.T) {
	store := testutil.SetupTestStore(t)
	appOrg := testutil.CreateTestAppOrganization(t, store, "http://localhost:3000/callback")
	apiKey := testutil.CreateTestAPIKey(t, store, appOrg.Environment, false)

	service := &Service{Store: store}
	ctx := testutil.WithAPIKeyContext(context.Background(), apiKey)

	req := connect.NewRequest(&ssoreadyv1.GetSAMLRedirectURLRequest{
		OrganizationExternalId: "invalid-org-id",
	})

	_, err := service.GetSAMLRedirectURL(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "organization")
}

func TestRedeemSAMLAccessCode_ValidCode(t *testing.T) {
	_ = testutil.SetupTestStore(t)
	// service := &Service{Store: store}

	t.Skip("Requires database setup with SAML flow and access code")
}

func TestRedeemSAMLAccessCode_InvalidCode(t *testing.T) {
	store := testutil.SetupTestStore(t)
	appOrg := testutil.CreateTestAppOrganization(t, store, "http://localhost:3000/callback")
	apiKey := testutil.CreateTestAPIKey(t, store, appOrg.Environment, false)

	service := &Service{Store: store}
	ctx := testutil.WithAPIKeyContext(context.Background(), apiKey)

	req := connect.NewRequest(&ssoreadyv1.RedeemSAMLAccessCodeRequest{
		SamlAccessCode: "invalid-code",
	})

	_, err := service.RedeemSAMLAccessCode(ctx, req)
	require.Error(t, err)
	// Invalid code format returns invalid_argument error with prefix check message
	assert.Contains(t, err.Error(), "does not have expected prefix", "or another validation error")
}

func TestRedeemSAMLAccessCode_AlreadyRedeemed(t *testing.T) {
	_ = testutil.SetupTestStore(t)
	// service := &Service{Store: store}

	t.Skip("Requires database setup with already-redeemed access code")
}

func TestParseSAMLMetadata_ValidURL(t *testing.T) {
	// Load test metadata from testdata (find project root)
	projectRoot := getProjectRoot()
	metadataPath := fmt.Sprintf("%s/internal/saml/testdata/assertions/okta/metadata.xml", projectRoot)
	metadata, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	// Create test server that returns the metadata
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(metadata)
	}))
	defer server.Close()

	service := &Service{
		SAMLMetadataHTTPClient: &http.Client{},
	}

	req := connect.NewRequest(&ssoreadyv1.ParseSAMLMetadataRequest{
		Url: server.URL,
	})

	resp, err := service.ParseSAMLMetadata(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.IdpRedirectUrl)
	assert.NotEmpty(t, resp.Msg.IdpEntityId)
	assert.NotEmpty(t, resp.Msg.IdpCertificate)

	// Verify certificate is PEM-encoded
	block, _ := pem.Decode([]byte(resp.Msg.IdpCertificate))
	require.NotNil(t, block)
	assert.Equal(t, "CERTIFICATE", block.Type)
}

func TestParseSAMLMetadata_InvalidURL(t *testing.T) {
	service := &Service{
		SAMLMetadataHTTPClient: &http.Client{},
	}

	req := connect.NewRequest(&ssoreadyv1.ParseSAMLMetadataRequest{
		Url: "http://invalid-url-that-does-not-exist.example.com/metadata.xml",
	})

	_, err := service.ParseSAMLMetadata(context.Background(), req)
	require.Error(t, err)
}

func TestParseSAMLMetadata_NetworkError(t *testing.T) {
	service := &Service{
		SAMLMetadataHTTPClient: &http.Client{},
	}

	req := connect.NewRequest(&ssoreadyv1.ParseSAMLMetadataRequest{
		Url: "http://localhost:99999/nonexistent",
	})

	_, err := service.ParseSAMLMetadata(context.Background(), req)
	require.Error(t, err)
}

func TestParseSAMLMetadata_MalformedMetadata(t *testing.T) {
	// Create test server that returns invalid XML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte("not valid xml"))
	}))
	defer server.Close()

	service := &Service{
		SAMLMetadataHTTPClient: &http.Client{},
	}

	req := connect.NewRequest(&ssoreadyv1.ParseSAMLMetadataRequest{
		Url: server.URL,
	})

	_, err := service.ParseSAMLMetadata(context.Background(), req)
	require.Error(t, err)
	// ParseMetadata returns XML parsing errors (not ValidateError)
	// The error will be from xml.Unmarshal, which typically includes "EOF" or syntax errors
	// Just verify it's an error - the exact message depends on XML parsing implementation
	assert.NotEmpty(t, err.Error())
}

func TestParseSAMLMetadata_HTTPError(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	service := &Service{
		SAMLMetadataHTTPClient: &http.Client{},
	}

	req := connect.NewRequest(&ssoreadyv1.ParseSAMLMetadataRequest{
		Url: server.URL,
	})

	_, err := service.ParseSAMLMetadata(context.Background(), req)
	require.Error(t, err)
	// Should handle HTTP errors appropriately
}

func TestParseSAMLMetadata_EmptyResponse(t *testing.T) {
	// Create test server that returns empty body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		// Empty body
	}))
	defer server.Close()

	service := &Service{
		SAMLMetadataHTTPClient: &http.Client{},
	}

	req := connect.NewRequest(&ssoreadyv1.ParseSAMLMetadataRequest{
		Url: server.URL,
	})

	_, err := service.ParseSAMLMetadata(context.Background(), req)
	require.Error(t, err)
}

func TestParseSAMLMetadata_WithRealTestData(t *testing.T) {
	// Test with all available test metadata files
	testProviders := []string{"okta", "google", "adfs", "jumpcloud", "keycloak", "ping"}

	for _, provider := range testProviders {
		t.Run(provider, func(t *testing.T) {
			projectRoot := getProjectRoot()
			metadataPath := fmt.Sprintf("%s/internal/saml/testdata/assertions/%s/metadata.xml", projectRoot, provider)
			metadata, err := os.ReadFile(metadataPath)
			require.NoError(t, err)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.Write(metadata)
			}))
			defer server.Close()

			service := &Service{
				SAMLMetadataHTTPClient: &http.Client{},
			}

			req := connect.NewRequest(&ssoreadyv1.ParseSAMLMetadataRequest{
				Url: server.URL,
			})

			resp, err := service.ParseSAMLMetadata(context.Background(), req)
			require.NoError(t, err, "failed to parse metadata for %s", provider)
			assert.NotEmpty(t, resp.Msg.IdpRedirectUrl, "IdP redirect URL should not be empty for %s", provider)
			assert.NotEmpty(t, resp.Msg.IdpEntityId, "IdP entity ID should not be empty for %s", provider)
			assert.NotEmpty(t, resp.Msg.IdpCertificate, "IdP certificate should not be empty for %s", provider)
		})
	}
}
