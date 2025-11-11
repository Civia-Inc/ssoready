package testutil

import (
	"context"

	"github.com/Civia-Inc/ssoready/internal/authn"
	"github.com/Civia-Inc/ssoready/internal/store/idformat"
	"github.com/google/uuid"
)

// WithAPIKeyContext adds API key authentication to the context
func WithAPIKeyContext(ctx context.Context, apiKey *TestAPIKey) context.Context {
	return authn.NewContext(ctx, authn.ContextData{
		APIKey: &authn.APIKeyData{
			AppOrgID: apiKey.AppOrgID,
			EnvID:    idformat.Environment.Format(apiKey.EnvironmentID),
			APIKeyID: idformat.APIKey.Format(apiKey.ID),
		},
	})
}

// WithAdminContext adds admin access token authentication to the context
func WithAdminContext(ctx context.Context, orgID uuid.UUID, canManageSAML, canManageSCIM bool) context.Context {
	return authn.NewContext(ctx, authn.ContextData{
		AdminAccessToken: &authn.AdminAccessTokenData{
			OrganizationID: orgID,
			CanManageSAML:  canManageSAML,
			CanManageSCIM:  canManageSCIM,
		},
	})
}
