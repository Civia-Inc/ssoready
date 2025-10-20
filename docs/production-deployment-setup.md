# Production Deployment Setup

## Overview

Configure production deployment infrastructure on AWS using ECS Fargate, RDS PostgreSQL, and automated GitHub Actions deployment. This plan removes unnecessary features (custom domains, public Docker, email sending) and focuses on core deployment.

## Prerequisites Information Needed

Before starting, you'll need to decide on:

1. **AWS Account ID** - Your production AWS account
2. **Domain Name** - Your production domain (e.g., `yourdomain.com`)
3. **S3 Bucket Name** - For admin logos (e.g., `yourorg-prod-admin-logos`)
4. **AWS Region** - Keep `us-east-2` or change to your preferred region
5. **Google OAuth Client ID** - From Google Cloud Console
6. **Sentry DSN** (optional) - For error tracking or set to empty string

## Phase 1: Update GitHub Actions Workflows

### 1.1 Update `.github/workflows/deploy-prod.yaml`

- Replace AWS account ID in `ECR_REGISTRY_URL` (line 8)
- Change `environment: dev-ucarion` to `environment: production` (lines 15, 58)
- Keep AWS region `us-east-2` or update throughout

### 1.2 Delete `.github/workflows/publish-docker-images.yaml`

- Not needed since you're not publishing public Docker images
- Deployment uses private ECR

## Phase 2: Update ECS Task Definitions

### 2.1 Update `cmd/api/task-definition-prod.json`

Replace the following:

- Line 118: `taskRoleArn` - Update account ID `381491982249` to yours
- Line 119: `executionRoleArn` - Update account ID to yours
- Line 21: `API_LOAD_AWS_SECRET_ID` value stays `api`
- Lines 24-29: Update or remove Sentry DSN and environment
- Lines 36-57: Update all URL environment variables to your domain:
  - `API_DEFAULT_AUTH_URL` → `https://auth.yourdomain.com`
  - `API_DEFAULT_ADMIN_SETUP_URL` → `https://admin.yourdomain.com/setup`
  - `API_EMAIL_VERIFICATION_ENDPOINT` → Remove (not using email)
  - `API_GOOGLE_OAUTH_CLIENT_ID` → Your Google OAuth client ID
  - `API_MICROSOFT_OAUTH_*` → Remove (not using)
- Lines 64-93: **Remove all custom domain environment variables**:
  - Remove `API_EMAIL_CHALLENGE_FROM`
  - Remove `API_EMAIL_VERIFICATION_ENDPOINT`
  - Remove `API_CUSTOM_AUTH_DOMAIN_CLOUDFLARE_*`
  - Remove `API_CUSTOM_ADMIN_DOMAIN_CLOUDFLARE_*`
  - Remove `API_FLYIO_*` variables (all 4)
- Line 96-98: Update `API_ADMIN_LOGOS_S3_BUCKET_NAME` to your bucket name

### 2.2 Update `cmd/auth/task-definition-prod.json`

Replace the following:

- Line 62: `taskRoleArn` - Update account ID to yours
- Line 63: `executionRoleArn` - Update account ID to yours
- Lines 24-29: Update or remove Sentry configuration
- Lines 36-42: Update URLs to your domain:
  - `AUTH_DEFAULT_ADMIN_TEST_MODE_URL` → `https://admin.yourdomain.com/test-mode`
  - `AUTH_BASE_URL` → `https://auth.yourdomain.com`

## Phase 3: AWS Infrastructure Setup

### 3.1 Create IAM Roles

**Create `ecsTaskExecutionRole`** with:

- AWS managed policy: `AmazonECSTaskExecutionRolePolicy`
- Custom inline policy for Secrets Manager:
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "secretsmanager:GetSecretValue"
    ],
    "Resource": [
      "arn:aws:secretsmanager:us-east-2:YOUR_ACCOUNT_ID:secret:api-*",
      "arn:aws:secretsmanager:us-east-2:YOUR_ACCOUNT_ID:secret:auth-*"
    ]
  }]
}
```


**Create `api` task role** with inline policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "arn:aws:secretsmanager:us-east-2:YOUR_ACCOUNT_ID:secret:api-*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:s3:::YOUR_BUCKET_NAME/*"
    }
  ]
}
```

**Create `auth` task role** with inline policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "secretsmanager:GetSecretValue"
    ],
    "Resource": "arn:aws:secretsmanager:us-east-2:YOUR_ACCOUNT_ID:secret:auth-*"
  }]
}
```

**Create IAM user for GitHub Actions** with policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:PutImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ecs:UpdateService",
        "ecs:DescribeServices",
        "ecs:DescribeTaskDefinition",
        "ecs:RegisterTaskDefinition"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "iam:PassRole",
      "Resource": [
        "arn:aws:iam::YOUR_ACCOUNT_ID:role/api",
        "arn:aws:iam::YOUR_ACCOUNT_ID:role/auth",
        "arn:aws:iam::YOUR_ACCOUNT_ID:role/ecsTaskExecutionRole"
      ]
    }
  ]
}
```

### 3.2 Create ECR Repositories

```bash
aws ecr create-repository --repository-name api --region us-east-2
aws ecr create-repository --repository-name auth --region us-east-2
```

### 3.3 Create RDS Aurora PostgreSQL Cluster

**Specifications**:

- Engine: PostgreSQL 15+
- Capacity: Aurora Serverless v2 (recommended) or provisioned
- Enable IAM database authentication
- VPC with private subnets
- Security group allowing port 5432 from ECS tasks

**Create database user**:

```sql
CREATE USER db_user;
GRANT rds_iam TO db_user;
GRANT ALL PRIVILEGES ON DATABASE postgres TO db_user;
```

### 3.4 Generate Cryptographic Secrets

Run these commands to generate required secrets:

```bash
# Pagination encoding (64-char hex)
openssl rand -hex 32

# SAML state signing (64-char hex)
openssl rand -hex 32

# OAuth ID token signing key (RSA 4096-bit, base64-encoded)
openssl genrsa 4096 | base64 -w 0
```

### 3.5 Create AWS Secrets Manager Secrets

**Secret name: `api`** (JSON):

```json
{
  "API_DB": "postgres://db_user@YOUR_RDS_ENDPOINT:5432/postgres?sslmode=require",
  "API_PAGE_ENCODING_VALUE": "PASTE_64_CHAR_HEX_HERE",
  "API_SAML_STATE_SIGNING_KEY": "PASTE_64_CHAR_HEX_HERE",
  "API_OAUTH_ID_TOKEN_PRIVATE_KEY_BASE64": "PASTE_BASE64_RSA_KEY_HERE",
  "API_GOOGLE_OAUTH_CLIENT_SECRET": "YOUR_GOOGLE_OAUTH_CLIENT_SECRET"
}
```

**Note**: Remove these unused keys from secret:

- `API_RESEND_API_KEY` (no email)
- `API_CLOUDFLARE_API_KEY` (no custom domains)
- `API_FLYIO_API_KEY` (no custom domains)
- `API_SEGMENT_WRITE_KEY` (optional analytics)
- `API_MICROSOFT_OAUTH_CLIENT_SECRET` (not using Microsoft)

**Secret name: `auth`** (JSON):

```json
{
  "AUTH_DB": "postgres://db_user@YOUR_RDS_ENDPOINT:5432/postgres?sslmode=require",
  "AUTH_PAGE_ENCODING_VALUE": "SAME_AS_API_PAGE_ENCODING",
  "AUTH_SAML_STATE_SIGNING_KEY": "SAME_AS_API_SAML_STATE_SIGNING",
  "AUTH_OAUTH_ID_TOKEN_PRIVATE_KEY_BASE64": "SAME_AS_API_OAUTH_KEY"
}
```

**Secret name: `psql`** (JSON, for migrations):

```json
{
  "DATABASE_URL_WRITE": "postgres://db_user@YOUR_RDS_ENDPOINT:5432/postgres?sslmode=require"
}
```

### 3.6 Create S3 Bucket

```bash
aws s3 mb s3://YOUR_BUCKET_NAME --region us-east-2
```

Configure CORS for browser uploads:

```json
[{
  "AllowedHeaders": ["*"],
  "AllowedMethods": ["GET", "PUT", "POST"],
  "AllowedOrigins": ["https://admin.yourdomain.com"],
  "ExposeHeaders": []
}]
```

### 3.7 Create VPC Infrastructure (if not exists)

- VPC with public and private subnets (at least 2 AZs)
- NAT Gateway in public subnet
- Internet Gateway
- Route tables configured
- Security groups:
  - ALB security group (allow 443 from internet)
  - ECS security group (allow 8080 from ALB)
  - RDS security group (allow 5432 from ECS)

### 3.8 Create Application Load Balancer

**ALB Configuration**:

- Scheme: Internet-facing
- Listeners: HTTPS (443) with ACM certificate
- Target groups:
  - `api-tg`: Port 8080, health check `/internal/health`
  - `auth-tg`: Port 8080, health check `/internal/health`

**Routing Rules**:

- `api.yourdomain.com` → `api-tg`
- `auth.yourdomain.com` → `auth-tg`

### 3.9 Create ECS Cluster

```bash
aws ecs create-cluster --cluster-name main --region us-east-2
```

### 3.10 Run Database Migrations

Update `bin/migrate` to add your prod environment, then:

```bash
./bin/migrate prod up
```

### 3.11 Create ECS Services

**For API service**:

```bash
aws ecs create-service \
  --cluster main \
  --service-name api \
  --task-definition api:1 \
  --desired-count 1 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[SUBNET_IDS],securityGroups=[SG_ID],assignPublicIp=DISABLED}" \
  --load-balancers "targetGroupArn=TARGET_GROUP_ARN,containerName=api,containerPort=8080"
```

**For AUTH service** (similar command, different service name)

## Phase 4: Frontend Deployment

### 4.1 Set Up Netlify Sites

**For `app` frontend**:

1. Create new Netlify site
2. Build command: `npm run build-docker`
3. Publish directory: `public`
4. Build directory: `app/`
5. Environment variables:

   - `APP_API_URL=https://api.yourdomain.com/internal/connect`
   - `APP_APP_URL=https://app.yourdomain.com`
   - `APP_PUBLIC_API_URL=https://api.yourdomain.com`

**For `admin` frontend**:

1. Create new Netlify site
2. Build command: `npm run build-docker`
3. Publish directory: `public`
4. Build directory: `admin/`
5. Environment variables:

   - `ADMIN_API_URL=https://api.yourdomain.com/internal/connect`

### 4.2 Configure DNS

- `api.yourdomain.com` → ALB DNS (CNAME)
- `auth.yourdomain.com` → ALB DNS (CNAME)
- `app.yourdomain.com` → Netlify (per their instructions)
- `admin.yourdomain.com` → Netlify (per their instructions)

## Phase 5: GitHub Secrets Configuration

Add these secrets to GitHub repository (Settings → Secrets and Variables → Actions):

- `AWS_ACCESS_KEY_ID_PROD` - IAM user access key
- `AWS_SECRET_ACCESS_KEY_PROD` - IAM user secret key

## Phase 6: Clean Up Unnecessary Code

### 6.1 Update `cmd/api/main.go`

Remove unused configuration fields (lines 58-71):

- `ResendAPIKey`
- `EmailChallengeFrom`
- `EmailVerificationEndpoint`
- `CloudflareAPIKey`
- `CustomAuthDomainCloudflareZoneID`
- `CustomAuthDomainCloudflareCNameValue`
- `CustomAdminDomainCloudflareZoneID`
- `CustomAdminDomainCloudflareCNameValue`
- `FlyioAPIKey`
- `FlyioAuthProxyAppID`
- `FlyioAuthProxyAppCNAMEValue`
- `FlyioAdminProxyAppID`
- `FlyioAdminProxyAppCNAMEValue`

Also remove the corresponding service initializations (lines 120-159).

### 6.2 Remove Fly.io Proxy Configurations

These directories/files are not needed:

- `cmd/authproxy/` (entire directory)
- `cmd/adminproxy/` (entire directory)

## Testing & Validation

### Test Checklist:

1. ✓ ECS tasks can fetch secrets from Secrets Manager
2. ✓ ECS tasks can connect to RDS database
3. ✓ API service health check responds: `curl https://api.yourdomain.com/internal/health`
4. ✓ Auth service health check responds: `curl https://auth.yourdomain.com/internal/health`
5. ✓ Frontend apps can connect to API
6. ✓ Google OAuth login works
7. ✓ Admin logo upload to S3 works
8. ✓ GitHub Actions deployment succeeds on push to `main`

## Rollout Strategy

1. **Initial Deploy**: Manually push first Docker images to ECR and create services
2. **Verify**: Test all functionality in production
3. **Enable CI/CD**: Merge updated workflows to enable automated deployment
4. **Monitor**: Check CloudWatch logs and Sentry for errors

## Alternative: Replace Netlify with AWS (Optional)

If you prefer to host frontends on AWS instead of Netlify, see the appendix document `docs/aws-frontend-hosting.md` for detailed instructions on using S3 + CloudFront.
