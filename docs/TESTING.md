# Testing Strategy for SSOReady

## Current Test Coverage

### ✅ Backend Tests (Go)
- **SAML functionality**: Comprehensive tests for assertion validation, XML processing, canonicalization
- **SCIM functionality**: Tests for patch operations and path parsing
- **Utility functions**: Email parsing, pagination tokens, user resource conversion
- **Test data**: Real SAML assertions from various providers (Okta, Google, ADFS, etc.)

### ❌ Missing Test Coverage
- **API endpoints**: No tests for HTTP handlers and business logic
- **Database operations**: No integration tests with PostgreSQL
- **Authentication flows**: No tests for OAuth/SAML login flows
- **Frontend components**: No React component tests
- **End-to-end tests**: No full user journey tests

## Frontend Testing Setup Recommendations

### Current State
- No frontend tests implemented
- TypeScript checking and build verification only
- ESLint and Prettier configured but not enforced in CI

### Recommended Frontend Testing Stack

#### 1. Unit Testing
```bash
# Add to package.json
npm install --save-dev @testing-library/react @testing-library/jest-dom jest jest-environment-jsdom
```

**Jest Configuration** (`jest.config.js`):
```javascript
module.exports = {
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts'],
  moduleNameMapping: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
  testMatch: ['**/__tests__/**/*.(test|spec).(ts|tsx)'],
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/index.tsx',
  ],
};
```

#### 2. Component Testing
```bash
# Add to package.json
npm install --save-dev @testing-library/user-event @testing-library/react-hooks
```

**Example test** (`src/components/Button.test.tsx`):
```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from './Button';

describe('Button', () => {
  it('renders with correct text', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('calls onClick when clicked', () => {
    const handleClick = jest.fn();
    render(<Button onClick={handleClick}>Click me</Button>);
    fireEvent.click(screen.getByText('Click me'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });
});
```

#### 3. Integration Testing
```bash
# Add to package.json
npm install --save-dev @testing-library/react @testing-library/jest-dom msw
```

**MSW Setup** (`src/mocks/handlers.ts`):
```typescript
import { rest } from 'msw';

export const handlers = [
  rest.get('/api/users', (req, res, ctx) => {
    return res(
      ctx.json([
        { id: 1, name: 'John Doe', email: 'john@example.com' }
      ])
    );
  }),
];
```

#### 4. E2E Testing (Optional)
```bash
# Add to package.json
npm install --save-dev playwright @playwright/test
```

**Playwright Config** (`playwright.config.ts`):
```typescript
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:8082',
    trace: 'on-first-retry',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
    { name: 'webkit', use: { ...devices['Desktop Safari'] } },
  ],
});
```

## Backend Testing Enhancements

### 1. API Endpoint Testing
```go
// internal/apiservice/saml_test.go
func TestGetSamlRedirectUrl(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Create test organization
    org := createTestOrganization(t, db)

    // Test handler
    handler := NewSamlHandler(db)
    req := &ssoreadyv1.GetSamlRedirectUrlRequest{
        OrganizationExternalId: org.ExternalID,
    }

    resp, err := handler.GetSamlRedirectUrl(context.Background(), req)
    require.NoError(t, err)
    assert.NotEmpty(t, resp.RedirectUrl)
}
```

### 2. Database Integration Tests
```go
// internal/store/integration_test.go
func TestUserCRUD(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Test user creation
    user := &User{
        Email: "test@example.com",
        Name:  "Test User",
    }

    err := db.CreateUser(user)
    require.NoError(t, err)
    assert.NotZero(t, user.ID)

    // Test user retrieval
    retrieved, err := db.GetUserByEmail("test@example.com")
    require.NoError(t, err)
    assert.Equal(t, user.Email, retrieved.Email)
}
```

### 3. Authentication Flow Tests
```go
// internal/authservice/oauth_test.go
func TestGoogleOAuthFlow(t *testing.T) {
    // Mock Google OAuth responses
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return mock Google OAuth response
    }))
    defer server.Close()

    // Test OAuth flow
    authService := NewAuthService(server.URL)
    token, err := authService.ExchangeCodeForToken("test-code")
    require.NoError(t, err)
    assert.NotEmpty(t, token)
}
```

## Testing Best Practices

### 1. Test Organization
```
tests/
├── unit/           # Unit tests for individual functions
├── integration/    # Tests requiring external dependencies
├── e2e/           # End-to-end user journey tests
└── fixtures/      # Test data and mock responses
```

### 2. Test Data Management
- Use factories for creating test data
- Clean up test data after each test
- Use database transactions for isolation
- Mock external services (OAuth providers, email services)

### 3. Coverage Goals
- **Backend**: 80%+ code coverage
- **Frontend**: 70%+ component coverage
- **Critical paths**: 100% coverage (SAML/SCIM flows)

### 4. CI/CD Integration
- Run tests on every PR
- Require all tests to pass before merge
- Generate coverage reports
- Upload test artifacts
- Security scanning on every build

## Next Steps

1. **Immediate** (Week 1):
   - Set up frontend testing framework
   - Add basic component tests for critical components
   - Implement API endpoint tests

2. **Short-term** (Month 1):
   - Add database integration tests
   - Implement authentication flow tests
   - Set up E2E testing for critical user journeys

3. **Long-term** (Quarter 1):
   - Achieve target coverage goals
   - Implement performance testing
   - Add load testing for SAML/SCIM endpoints

## Running Tests Locally

### Backend Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package
go test ./internal/saml/...

# Run with race detection
go test -race ./...
```

### Frontend Tests (when implemented)
```bash
# Run all tests
npm test

# Run with coverage
npm test -- --coverage

# Run in watch mode
npm test -- --watch

# Run E2E tests
npx playwright test
```

## Monitoring Test Health

- **Test execution time**: Monitor for slow tests
- **Flaky tests**: Track and fix unreliable tests
- **Coverage trends**: Ensure coverage doesn't decrease
- **Security scan results**: Address vulnerabilities promptly
