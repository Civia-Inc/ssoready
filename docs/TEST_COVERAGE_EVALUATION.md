# Go Test Coverage Evaluation for Auth and API Services

## Executive Summary

This document evaluates the current test coverage for the Go backend services (auth and API), identifies gaps, and provides recommendations for improving test quality and coverage.

**Current Coverage**: ~15-20% of production code
**Test Files**: 10 test files found
**Critical Gap**: **No tests for API service handlers or Auth service HTTP handlers**

---

## What IS Currently Tested

### 1. SAML Core Functionality ✅
**Location**: `internal/saml/validate_test.go`

**Coverage:**
- ✅ SAML assertion validation with real test data from Okta, Google, ADFS, etc.
- ✅ Good assertion validation (signatures, certificates, entity IDs)
- ✅ Bad assertion detection:
  - Unsigned assertions
  - Expired assertions (early and late)
  - Bad IDP/SP entity IDs
  - Bad signature/digest algorithms
  - Bad certificates
  - Malformed assertions (UTF-8 errors)

**Quality**: Excellent - uses real-world SAML assertions and covers edge cases

### 2. SCIM Patch Operations ✅
**Location**: `internal/scimpatch/scimpatch_test.go`

**Coverage:**
- ✅ Comprehensive SCIM PATCH operation tests
- ✅ Replace, Add operations with various paths
- ✅ Filter expressions (eq, ne, co, sw, ew, pr, gt, ge, lt, le)
- ✅ Enterprise user schema handling
- ✅ Array manipulation with filters
- ✅ Error cases (invalid paths, unsupported operations)

**Quality**: Excellent - very thorough test coverage with edge cases

### 3. SCIM Resource Conversion ✅
**Location**: `internal/authservice/scim_test.go`

**Coverage:**
- ✅ SCIM user to resource conversion (manager reference handling)
- ✅ SCIM filter regex parsing (userName and email.value filters)

**Quality**: Good - covers specific SCIM formatting requirements

### 4. Utility Functions ✅
**Locations**:
- `internal/emailaddr/emailaddr_test.go`
- `internal/pagetoken/pagetoken_test.go`
- `internal/saml/c14n/c14n_test.go`
- `internal/saml/uxml/parse_test.go`
- `internal/saml/sortattr/sortattr_test.go`

**Coverage:**
- ✅ Email address parsing (including Microsoft #EXT# format)
- ✅ Pagination token encoding/decoding
- ✅ XML canonicalization
- ✅ XML parsing
- ✅ XML attribute sorting

**Quality**: Good - covers utility functions well

---

## What IS NOT Tested (Critical Gaps)

### 1. API Service Handlers ❌
**Location**: `internal/apiservice/`

**Zero test coverage** for all RPC handlers:
- ❌ `GetSAMLRedirectURL` - No tests
- ❌ `RedeemSAMLAccessCode` - No tests
- ❌ `ParseSAMLMetadata` - No tests
- ❌ `SignIn` (Google, Microsoft, Email) - No tests
- ❌ `VerifyEmail` - No tests
- ❌ `SignOut` - No tests
- ❌ `ListAPIKeys` / `GetAPIKey` / `CreateAPIKey` / `DeleteAPIKey` - No tests
- ❌ `ListEnvironments` / `GetEnvironment` / `CreateEnvironment` / `UpdateEnvironment` - No tests
- ❌ `ListSAMLConnections` / `GetSAMLConnection` / `CreateSAMLConnection` / `UpdateSAMLConnection` - No tests
- ❌ `ListSCIMDirectories` / `GetSCIMDirectory` / `CreateSCIMDirectory` / `UpdateSCIMDirectory` - No tests
- ❌ `ListSCIMUsers` / `GetSCIMUser` - No tests
- ❌ `ListSCIMGroups` / `GetSCIMGroup` - No tests
- ❌ All admin management endpoints - No tests

**Impact**: **CRITICAL** - No validation that the primary API interface works correctly

### 2. Auth Service HTTP Handlers ❌
**Location**: `internal/authservice/`

**Zero test coverage** for HTTP endpoints:
- ❌ `samlInit` (`GET /v1/saml/{saml_conn_id}/init`) - No tests
- ❌ `samlAcs` (`POST /v1/saml/{saml_conn_id}/acs`) - No tests
- ❌ `oauthOpenIDConfiguration` (`GET /v1/oauth/.well-known/openid-configuration`) - No tests
- ❌ `oauthAuthorize` (`GET /v1/oauth/authorize`) - No tests
- ❌ `oauthToken` (`POST /v1/oauth/token`) - No tests
- ❌ `oauthUserinfo` (`GET /v1/oauth/userinfo`) - No tests
- ❌ `oauthJWKS` (`GET /v1/oauth/jwks`) - No tests
- ❌ `scimListUsers` (`GET /v1/scim/{scim_directory_id}/Users`) - No tests
- ❌ `scimGetUser` (`GET /v1/scim/{scim_directory_id}/Users/{scim_user_id}`) - No tests
- ❌ `scimCreateUser` (`POST /v1/scim/{scim_directory_id}/Users`) - No tests
- ❌ `scimUpdateUser` (`PUT /v1/scim/{scim_directory_id}/Users/{scim_user_id}`) - No tests
- ❌ `scimPatchUser` (`PATCH /v1/scim/{scim_directory_id}/Users/{scim_user_id}`) - No tests
- ❌ `scimDeleteUser` (`DELETE /v1/scim/{scim_directory_id}/Users/{scim_user_id}`) - No tests
- ❌ `scimListGroups` (`GET /v1/scim/{scim_directory_id}/Groups`) - No tests
- ❌ `scimGetGroup` (`GET /v1/scim/{scim_directory_id}/Groups/{scim_group_id}`) - No tests
- ❌ `scimCreateGroup` (`POST /v1/scim/{scim_directory_id}/Groups`) - No tests
- ❌ `scimUpdateGroup` (`PUT /v1/scim/{scim_directory_id}/Groups/{scim_group_id}`) - No tests
- ❌ `scimPatchGroup` (`PATCH /v1/scim/{scim_directory_id}/Groups/{scim_group_id}`) - No tests
- ❌ `scimDeleteGroup` (`DELETE /v1/scim/{scim_directory_id}/Groups/{scim_group_id}`) - No tests

**Impact**: **CRITICAL** - No validation that authentication flows work end-to-end

### 3. Error Handling and Edge Cases ❌

**Missing Coverage:**
- ❌ HTTP error responses (400, 401, 404, 500)
- ❌ Invalid input validation
- ❌ Malformed request bodies
- ❌ Authentication failures
- ❌ Database errors
- ❌ External service failures (Google OAuth, Microsoft OAuth, Resend email)
- ❌ Timeout handling
- ❌ Rate limiting
- ❌ Concurrent request handling

### 4. Integration Tests ❌

**Missing Coverage:**
- ❌ Database integration tests
- ❌ Full authentication flow tests (end-to-end)
- ❌ SAML flow integration tests
- ❌ SCIM provisioning flow tests
- ❌ OAuth flow integration tests
- ❌ Cross-service communication tests

### 5. Security Testing ❌

**Missing Coverage:**
- ❌ Token validation
- ❌ Authorization checks
- ❌ SQL injection prevention
- ❌ XSS prevention
- ❌ CSRF protection
- ❌ Rate limiting
- ❌ Input sanitization

---

## Recommended Additional Tests

### Priority 1: Critical Path Tests (Implement First)

#### A. API Service Handler Tests

**1. SAML Endpoints**
```go
// internal/apiservice/saml_test.go
func TestGetSAMLRedirectURL(t *testing.T) {
    // Test happy path
    // Test invalid organization
    // Test missing SAML connection
    // Test unconfigured SAML connection
}

func TestRedeemSAMLAccessCode(t *testing.T) {
    // Test valid code
    // Test invalid code
    // Test already-redeemed code
    // Test expired code
}

func TestParseSAMLMetadata(t *testing.T) {
    // Test valid metadata URL
    // Test invalid URL
    // Test network error
    // Test malformed metadata
}
```

**2. Authentication Endpoints**
```go
// internal/apiservice/signin_test.go
func TestSignIn_GoogleCredential(t *testing.T) {
    // Test valid Google credential
    // Test invalid credential
    // Test expired credential
}

func TestSignIn_MicrosoftCode(t *testing.T) {
    // Test valid Microsoft code
    // Test invalid code
    // Test token exchange failure
}

func TestSignIn_EmailVerifyToken(t *testing.T) {
    // Test valid token
    // Test invalid token
    // Test expired token
}

func TestVerifyEmail(t *testing.T) {
    // Test email sent successfully
    // Test invalid email format
    // Test email service failure
}

func TestSignOut(t *testing.T) {
    // Test successful sign out
    // Test invalid session
}
```

**3. API Key Management**
```go
// internal/apiservice/api_keys_test.go
func TestCreateAPIKey(t *testing.T) {
    // Test successful creation
    // Test authorization check
    // Test validation
}

func TestListAPIKeys(t *testing.T) {
    // Test pagination
    // Test filtering
    // Test authorization
}
```

#### B. Auth Service HTTP Handler Tests

**1. SAML Handlers**
```go
// internal/authservice/saml_test.go
func TestSamlInit(t *testing.T) {
    // Test successful redirect to IdP
    // Test invalid SAML connection ID
    // Test bad state parameter
    // Test unconfigured connection
}

func TestSamlAcs(t *testing.T) {
    // Test successful assertion processing
    // Test invalid assertion
    // Test expired assertion
    // Test domain validation
    // Test duplicate assertion
    // Test error template rendering
}
```

**2. OAuth Handlers**
```go
// internal/authservice/oauth_test.go
func TestOAuthAuthorize(t *testing.T) {
    // Test successful authorization redirect
    // Test invalid client_id
    // Test missing parameters
    // Test unconfigured environment
}

func TestOAuthToken(t *testing.T) {
    // Test valid code exchange
    // Test invalid code
    // Test invalid client credentials
    // Test JWT token generation
}

func TestOAuthUserinfo(t *testing.T) {
    // Test valid token
    // Test invalid token
    // Test expired token
}

func TestOAuthJWKS(t *testing.T) {
    // Test JWKS endpoint returns public key
}
```

**3. SCIM Handlers**
```go
// internal/authservice/scim_test.go
func TestSCIMListUsers(t *testing.T) {
    // Test with filter
    // Test without filter
    // Test pagination
    // Test invalid bearer token
    // Test directory not found
}

func TestSCIMCreateUser(t *testing.T) {
    // Test successful creation
    // Test invalid email
    // Test domain validation
    // Test duplicate user
    // Test invalid bearer token
}

func TestSCIMPatchUser(t *testing.T) {
    // Test valid patch operations
    // Test invalid patch operations
    // Test email preservation
    // Test domain validation after patch
}
```

### Priority 2: Integration Tests

**1. Database Integration Tests**
```go
// internal/store/integration_test.go
func TestStore_UserCRUD(t *testing.T) {
    // Setup test database with transactions
    // Test user creation, retrieval, update, deletion
    // Test transaction rollback on error
}

func TestStore_SAMLFlow(t *testing.T) {
    // Test complete SAML flow from init to redeem
    // Test state management
    // Test assertion processing
}
```

**2. End-to-End Flow Tests**
```go
// tests/e2e/saml_flow_test.go
func TestE2E_SAMLAuthenticationFlow(t *testing.T) {
    // 1. Call GetSAMLRedirectURL
    // 2. Verify redirect URL
    // 3. Simulate IdP response
    // 4. Call RedeemSAMLAccessCode
    // 5. Verify user email returned
}

func TestE2E_SCIMProvisioningFlow(t *testing.T) {
    // 1. Create SCIM user via API
    // 2. Verify user in database
    // 3. Update user via SCIM
    // 4. Verify update
}
```

### Priority 3: Error Handling and Edge Cases

**1. Error Response Tests**
```go
func TestAPIErrorHandling(t *testing.T) {
    // Test 400 Bad Request
    // Test 401 Unauthorized
    // Test 404 Not Found
    // Test 500 Internal Server Error
    // Test proper error message format
}
```

**2. Input Validation Tests**
```go
func TestInputValidation(t *testing.T) {
    // Test invalid email formats
    // Test SQL injection attempts
    // Test XSS attempts
    // Test overly long inputs
    // Test missing required fields
}
```

**3. Concurrent Request Tests**
```go
func TestConcurrentRequests(t *testing.T) {
    // Test concurrent SAML flows
    // Test concurrent SCIM operations
    // Test race conditions
}
```

### Priority 4: Security Tests

**1. Authorization Tests**
```go
func TestAuthorization(t *testing.T) {
    // Test API key validation
    // Test session validation
    // Test organization access control
    // Test environment access control
}
```

**2. Security Vulnerability Tests**
```go
func TestSecurity(t *testing.T) {
    // Test token validation
    // Test bearer token handling
    // Test state parameter validation
    // Test replay attack prevention
}
```

---

## Test Infrastructure Recommendations

### 1. Test Database Setup
```go
// internal/testutil/db.go
func SetupTestDB(t *testing.T) *store.Store {
    // Create temporary database
    // Run migrations
    // Return store instance
    // Cleanup on test completion
}
```

### 2. Test Fixtures
```go
// internal/testutil/fixtures.go
func CreateTestOrganization(t *testing.T, store *store.Store) *Organization
func CreateTestSAMLConnection(t *testing.T, store *store.Store) *SAMLConnection
func CreateTestSCIMDirectory(t *testing.T, store *store.Store) *SCIMDirectory
```

### 3. HTTP Test Helpers
```go
// internal/testutil/http.go
func MakeAuthRequest(t *testing.T, handler http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder
func MakeAPIRequest(t *testing.T, service *apiservice.Service, method, endpoint string, req interface{}) *httptest.ResponseRecorder
```

### 4. Mock External Services
```go
// internal/testutil/mocks.go
func MockGoogleOAuth(t *testing.T) *httptest.Server
func MockMicrosoftOAuth(t *testing.T) *httptest.Server
func MockEmailService(t *testing.T) *httptest.Server
```

---

## Testing Best Practices to Follow

1. **Test Organization**
   - Unit tests: `*_test.go` alongside source files
   - Integration tests: `*_integration_test.go`
   - E2E tests: `tests/e2e/` directory

2. **Test Data Management**
   - Use test fixtures/factories
   - Clean up after each test
   - Use database transactions for isolation
   - Use table-driven tests where appropriate

3. **Test Naming**
   - Use descriptive test names: `TestFunctionName_Scenario_ExpectedResult`
   - Example: `TestSignIn_GoogleCredential_ReturnsSessionToken`

4. **Coverage Goals**
   - Critical paths: 100% coverage (SAML, SCIM, OAuth)
   - API handlers: 80%+ coverage
   - Utility functions: 90%+ coverage

5. **CI/CD Integration**
   - Run all tests on every PR
   - Require 80%+ coverage for new code
   - Generate coverage reports
   - Run race condition tests

---

## Conclusion

The current test suite provides excellent coverage for **low-level SAML and SCIM operations** but has **critical gaps** in testing the **HTTP/RPC handlers** that clients interact with. The highest priority should be adding tests for:

1. **API service RPC handlers** (saml.go, signin.go, etc.)
2. **Auth service HTTP handlers** (samlInit, samlAcs, oauth handlers, SCIM handlers)
3. **Integration tests** for complete authentication flows

Once these are in place, focus on error handling, security, and edge cases to achieve comprehensive test coverage.
