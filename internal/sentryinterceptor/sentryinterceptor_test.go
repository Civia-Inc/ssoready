package sentryinterceptor

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/Civia-Inc/ssoready/internal/authn"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransport is a Sentry transport that captures events for testing
type mockTransport struct {
	events []*sentry.Event
}

func (t *mockTransport) Configure(options sentry.ClientOptions) {}
func (t *mockTransport) SendEvent(event *sentry.Event) {
	t.events = append(t.events, event)
}
func (t *mockTransport) Flush(timeout time.Duration) bool { return true }
func (t *mockTransport) Events() []*sentry.Event {
	return t.events
}
func (t *mockTransport) Reset() {
	t.events = nil
}

// setupTestContext creates a context with a Sentry hub that uses a mock transport
func setupTestContext(t *testing.T) (context.Context, *mockTransport) {
	t.Helper()
	transport := &mockTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Transport: transport,
		Dsn:       "https://test@test.ingest.sentry.io/test",
	})
	require.NoError(t, err)

	hub := sentry.NewHub(client, sentry.NewScope())
	ctx := sentry.SetHubOnContext(context.Background(), hub)
	return ctx, transport
}

// testRequest wraps a real Connect request to set the procedure
type testRequest struct {
	*connect.Request[struct{}]
	procedure string
}

func newTestRequest(procedure string) connect.AnyRequest {
	req := connect.NewRequest[struct{}](&struct{}{})
	// We use a wrapper to override Spec() to return our test procedure
	// This allows us to test the interceptor with a real Connect request
	return &testRequest{
		Request:   req,
		procedure: procedure,
	}
}

func (t *testRequest) Spec() connect.Spec {
	return connect.Spec{
		Procedure: t.procedure,
	}
}

func TestNewPreAuthentication_ExpectedErrorsNotLogged(t *testing.T) {
	testCases := []struct {
		name      string
		errorCode connect.Code
		shouldLog bool
	}{
		{
			name:      "CodeNotFound should not be logged",
			errorCode: connect.CodeNotFound,
			shouldLog: false,
		},
		{
			name:      "CodeUnauthenticated should not be logged",
			errorCode: connect.CodeUnauthenticated,
			shouldLog: false,
		},
		{
			name:      "CodeInternal should be logged",
			errorCode: connect.CodeInternal,
			shouldLog: true,
		},
		{
			name:      "CodeUnknown should be logged",
			errorCode: connect.CodeUnknown,
			shouldLog: true,
		},
		{
			name:      "CodeInvalidArgument should be logged",
			errorCode: connect.CodeInvalidArgument,
			shouldLog: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, transport := setupTestContext(t)
			req := newTestRequest("test.procedure")

			// Create a handler that returns the test error
			handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				return nil, connect.NewError(tc.errorCode, errors.New("test error"))
			}

			// Wrap with interceptor
			interceptor := NewPreAuthentication()
			wrapped := interceptor(handler)

			// Call the wrapped handler
			_, err := wrapped(ctx, req)
			require.Error(t, err)

			// Check if error was logged to Sentry
			events := transport.Events()
			if tc.shouldLog {
				assert.Greater(t, len(events), 0, "Expected error to be logged to Sentry")
				// Verify the error is in the events
				found := false
				for _, event := range events {
					if event.Exception != nil && len(event.Exception) > 0 {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected exception in Sentry event")
			} else {
				assert.Equal(t, 0, len(events), "Expected error should not be logged to Sentry")
			}

			transport.Reset()
		})
	}
}

func TestNewPreAuthentication_NonConnectErrorsAreLogged(t *testing.T) {
	ctx, transport := setupTestContext(t)
	req := newTestRequest("test.procedure")

	// Create a handler that returns a non-Connect error
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, errors.New("non-connect error")
	}

	// Wrap with interceptor
	interceptor := NewPreAuthentication()
	wrapped := interceptor(handler)

	// Call the wrapped handler
	_, err := wrapped(ctx, req)
	require.Error(t, err)

	// Non-Connect errors should always be logged
	events := transport.Events()
	assert.Greater(t, len(events), 0, "Non-Connect errors should be logged to Sentry")
}

func TestNewPreAuthentication_EndpointTagSet(t *testing.T) {
	ctx, transport := setupTestContext(t)
	req := newTestRequest("test.procedure")

	// Create a handler that returns an error
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, connect.NewError(connect.CodeInternal, errors.New("test error"))
	}

	// Wrap with interceptor
	interceptor := NewPreAuthentication()
	wrapped := interceptor(handler)

	// Call the wrapped handler
	_, err := wrapped(ctx, req)
	require.Error(t, err)

	// Verify error was logged and check that endpoint tag was set in the event
	events := transport.Events()
	assert.Greater(t, len(events), 0, "Error should be logged")

	// Check that the endpoint tag is in the event
	if len(events) > 0 {
		event := events[0]
		assert.Equal(t, "test.procedure", event.Tags["endpoint"], "Endpoint tag should be set in Sentry event")
	}
}

func TestNewPreAuthentication_ErrorDetailsAddedToContext(t *testing.T) {
	ctx, transport := setupTestContext(t)
	req := newTestRequest("test.procedure")

	// Create a handler that returns a Connect error with details
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		err := connect.NewError(connect.CodeInternal, errors.New("test error"))
		// Note: Adding details to Connect errors requires protobuf messages
		// For this test, we'll just verify the code path executes
		return nil, err
	}

	// Wrap with interceptor
	interceptor := NewPreAuthentication()
	wrapped := interceptor(handler)

	// Call the wrapped handler
	_, err := wrapped(ctx, req)
	require.Error(t, err)

	// Verify error was logged
	events := transport.Events()
	assert.Greater(t, len(events), 0, "Error should be logged")

	// The interceptor should have attempted to add error details to context
	// (even if there are no details in this case, the code path should execute)
	// We verify this by checking that the event was created successfully
}

func TestNewPreAuthentication_SuccessfulRequest(t *testing.T) {
	ctx, transport := setupTestContext(t)
	req := newTestRequest("test.procedure")

	// Create a handler that succeeds
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	}

	// Wrap with interceptor
	interceptor := NewPreAuthentication()
	wrapped := interceptor(handler)

	// Call the wrapped handler
	_, err := wrapped(ctx, req)
	require.NoError(t, err)

	// No errors should be logged
	events := transport.Events()
	assert.Equal(t, 0, len(events), "Successful requests should not log errors")
}

func TestNewPostAuthentication_SetsOrgIDFromAppSession(t *testing.T) {
	ctx, transport := setupTestContext(t)
	req := newTestRequest("test.procedure")

	orgID := uuid.New()
	userID := "user-123"

	// Create context with AppSession
	authnData := authn.ContextData{
		AppSession: &authn.AppSessionData{
			AppOrgID:  orgID,
			AppUserID: userID,
		},
	}
	ctx = authn.NewContext(ctx, authnData)

	// Create a handler that succeeds
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	}

	// Wrap with interceptor
	interceptor := NewPostAuthentication()
	wrapped := interceptor(handler)

	// Call the wrapped handler
	_, err := wrapped(ctx, req)
	require.NoError(t, err)

	// Trigger an event to capture the scope state
	hub := sentry.GetHubFromContext(ctx)
	hub.CaptureMessage("test message")

	// Check that org_id tag and user were set in the event
	events := transport.Events()
	require.Greater(t, len(events), 0, "Event should be captured")
	event := events[0]
	assert.Equal(t, orgID.String(), event.Tags["org_id"], "org_id tag should be set from AppSession")
	assert.Equal(t, userID, event.User.ID, "User ID should be set from AppSession")
}

func TestNewPostAuthentication_SetsOrgIDFromAPIKey(t *testing.T) {
	ctx, transport := setupTestContext(t)
	req := newTestRequest("test.procedure")

	orgID := uuid.New()

	// Create context with APIKey (no AppSession)
	authnData := authn.ContextData{
		APIKey: &authn.APIKeyData{
			AppOrgID: orgID,
			EnvID:    "env-123",
			APIKeyID: "key-123",
		},
	}
	ctx = authn.NewContext(ctx, authnData)

	// Create a handler that succeeds
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	}

	// Wrap with interceptor
	interceptor := NewPostAuthentication()
	wrapped := interceptor(handler)

	// Call the wrapped handler
	_, err := wrapped(ctx, req)
	require.NoError(t, err)

	// Trigger an event to capture the scope state
	hub := sentry.GetHubFromContext(ctx)
	hub.CaptureMessage("test message")

	// Check that org_id tag was set (but no user since there's no AppSession)
	events := transport.Events()
	require.Greater(t, len(events), 0, "Event should be captured")
	event := events[0]
	assert.Equal(t, orgID.String(), event.Tags["org_id"], "org_id tag should be set from APIKey")
	assert.Empty(t, event.User.ID, "User ID should not be set when there's no AppSession")
}

func TestNewPostAuthentication_SetsOrgIDFromSAMLOAuthClient(t *testing.T) {
	ctx, transport := setupTestContext(t)
	req := newTestRequest("test.procedure")

	orgID := uuid.New()

	// Create context with SAMLOAuthClient (no AppSession or APIKey)
	authnData := authn.ContextData{
		SAMLOAuthClient: &authn.SAMLOAuthClientData{
			AppOrgID:      orgID,
			EnvID:         "env-123",
			OAuthClientID: "client-123",
		},
	}
	ctx = authn.NewContext(ctx, authnData)

	// Create a handler that succeeds
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	}

	// Wrap with interceptor
	interceptor := NewPostAuthentication()
	wrapped := interceptor(handler)

	// Call the wrapped handler
	_, err := wrapped(ctx, req)
	require.NoError(t, err)

	// Trigger an event to capture the scope state
	hub := sentry.GetHubFromContext(ctx)
	hub.CaptureMessage("test message")

	// Check that org_id tag was set
	events := transport.Events()
	require.Greater(t, len(events), 0, "Event should be captured")
	event := events[0]
	assert.Equal(t, orgID.String(), event.Tags["org_id"], "org_id tag should be set from SAMLOAuthClient")
}

func TestNewPostAuthentication_NoAuthnData(t *testing.T) {
	ctx, transport := setupTestContext(t)
	req := newTestRequest("test.procedure")

	// Context with no authn data
	// Create a handler that succeeds
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	}

	// Wrap with interceptor
	interceptor := NewPostAuthentication()
	wrapped := interceptor(handler)

	// Call the wrapped handler
	_, err := wrapped(ctx, req)
	require.NoError(t, err)

	// Trigger an event to capture the scope state
	hub := sentry.GetHubFromContext(ctx)
	hub.CaptureMessage("test message")

	// Check that no org_id tag or user was set
	events := transport.Events()
	require.Greater(t, len(events), 0, "Event should be captured")
	event := events[0]
	_, hasOrgID := event.Tags["org_id"]
	assert.False(t, hasOrgID, "org_id tag should not be set when there's no authn data")
	assert.Empty(t, event.User.ID, "User ID should not be set when there's no authn data")
}

func TestNewPreAuthentication_SAMLAccessCodeNotFoundError(t *testing.T) {
	ctx, transport := setupTestContext(t)
	req := newTestRequest("test.procedure")

	// This simulates the "saml access code not found, or already used" error
	// which uses CodeNotFound
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("saml access code not found, or already used"))
	}

	// Wrap with interceptor
	interceptor := NewPreAuthentication()
	wrapped := interceptor(handler)

	// Call the wrapped handler
	_, err := wrapped(ctx, req)
	require.Error(t, err)

	// This error should NOT be logged to Sentry
	events := transport.Events()
	assert.Equal(t, 0, len(events), "SAMLAccessCodeNotFoundError should not be logged to Sentry")
}
