package testutil

import (
	"context"

	"github.com/Civia-Inc/ssoready/internal/authn"
	"github.com/Civia-Inc/ssoready/internal/store/idformat"
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
