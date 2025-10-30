# SSORead
 <a href="https://github.com/ssoready/ssoready/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue" /></a>

## What is SSOReady?

Fork of [SSOReady](https://ssoready.com) - an **open-source,
straightforward** way to add SAML and SCIM support to your product:

* **[SSOReady SAML](https://ssoready.com/docs/saml/saml-quickstart)**: Everything you need to add SAML ("Enterprise SSO") to your product today.
* **[SSOReady SCIM](https://ssoready.com/docs/scim/scim-quickstart)**: Everything you need to add SCIM ("Enterprise Directory Sync") to your product today.
* **[Self-serve Setup UI](https://ssoready.com/docs/idp-configuration/enabling-self-service-configuration-for-your-customers)**:
  A hosted UI your customers use to onboard themselves onto SAML and/or
  SCIM.

## Architecture

SSOReady consists of five main components that work together to provide SAML and SCIM functionality:

```mermaid
graph TB
    subgraph "External Systems"
        DevApp[Your Application<br/>using SSOReady SDK]
        IdP[Customer's Identity Provider<br/>Okta, Entra, Google, etc.]
        SCIMClient[Customer's SCIM Client<br/>Okta, Entra, Google, etc.]
    end

    subgraph "SSOReady System"
        subgraph "Frontend"
            Admin[Admin UI<br/>:8083<br/>React]
            App[Self-Serve UI<br/>:8082<br/>React]
        end

        subgraph "Backend Services"
            API[API Service<br/>:8080<br/>Go]
            Auth[Auth Service<br/>:8081<br/>Go]
        end

        DB[(PostgreSQL<br/>:5433)]
    end

    %% Developer Application interactions
    DevApp -->|"1. getSamlRedirectUrl()"| API
    API -->|"Returns redirect URL"| DevApp
    DevApp -->|"2. User redirected"| IdP
    IdP -->|"3. SAML Response"| Auth
    Auth -->|"4. Callback with samlAccessCode"| DevApp
    DevApp -->|"5. redeemSamlAccessCode()"| API

    %% SCIM interactions
    SCIMClient -->|"SCIM 2.0 API<br/>(Users, Groups)"| Auth
    DevApp -->|"listScimUsers()<br/>listScimGroups()"| API

    %% Admin UI interactions
    Admin -->|"Configure SAML/SCIM<br/>via Connect RPC"| API

    %% Self-Serve UI interactions
    App -->|"Customer setup<br/>via Connect RPC"| API

    %% Backend interactions
    API <-->|"Read/Write<br/>Organizations, Environments,<br/>SAML Connections,<br/>SCIM Directories"| DB
    Auth <-->|"Read Configs<br/>Write SAML Flows,<br/>SCIM Data"| DB

    API -.->|"Generates URLs for"| Auth

    classDef frontend fill:#e1f5ff,stroke:#0288d1,stroke-width:2px
    classDef backend fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    classDef database fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef external fill:#e8f5e9,stroke:#388e3c,stroke-width:2px

    class Admin,App frontend
    class API,Auth backend
    class DB database
    class DevApp,IdP,SCIMClient external
```

### Component Responsibilities

| Component | Purpose | Technology | URL |
|-----------|---------|------------|-----|
| **API Service** | REST API for managing organizations, environments, SAML connections, and SCIM directories. Handles all administrative operations and SDK requests. | Go + Connect RPC | identity-api.govai.com |
| **Auth Service** | Handles SAML authentication flows and SCIM provisioning endpoints. Processes SAML assertions from IdPs and serves as SCIM 2.0 server. | Go | identity-auth.govai.com |
| **App (Admin) UI** | Management interface for configuring SAML/SCIM settings, viewing logs, and managing organizations. Referred to as app in code. | React + TypeScript | identity-app.govai.com |
| **Self-Serve UI** | Customer-facing interface allowing your customers to configure their own SAML/SCIM connections without developer involvement. Referred to as admin in code. | React + TypeScript | identity-setup.govai.com |
| **PostgreSQL** | Stores all configuration data, SAML flows, SCIM user/group data, and audit logs. | PostgreSQL | - |

### Data Flow Examples

**SAML Authentication Flow:**
1. Your app calls `getSamlRedirectUrl()` via SDK → API Service
2. API returns IdP redirect URL
3. User is redirected to customer's IdP
4. IdP authenticates user and sends SAML assertion → Auth Service
5. Auth Service validates assertion and redirects to your callback URL with `samlAccessCode`
6. Your app calls `redeemSamlAccessCode()` → API Service returns user email

**SCIM Provisioning Flow:**
1. Customer's IdP pushes user/group data via SCIM 2.0 → Auth Service
2. Auth Service validates and stores in PostgreSQL
3. Your app periodically calls `listScimUsers()` → API Service returns synced users


## Local development

To run SSOReady locally for development or testing:

1. Clone this repository
2. Run `./bin/dev-setup` to set up your environment
3. Run `./bin/dev-seed` to create development user accounts (optional)
4. Run `./bin/dev-start` to start all services

See [DEVELOPMENT.md](./DEVELOPMENT.md) for complete setup instructions, architecture details, and troubleshooting guides.


## Security

If you have a security issue to report, please contact us at
security@govai.com.
