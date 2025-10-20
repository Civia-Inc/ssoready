# AWS Frontend Hosting (Alternative to Netlify)

This document explains how to replace Netlify with AWS services for hosting the `app` and `admin` frontend applications.

## Overview

Replace Netlify's functionality using:
- **Amazon S3** - Static file hosting
- **Amazon CloudFront** - CDN with global edge locations
- **AWS Certificate Manager (ACM)** - Free SSL certificates
- **CloudFront Functions** - SPA routing logic
- **GitHub Actions** - Build and deploy automation

## Architecture

```
User Request
    ↓
CloudFront CDN (HTTPS, Custom Domain)
    ↓
CloudFront Function (SPA Routing)
    ↓
S3 Bucket (Static Files)
```

## Implementation Steps

### 1. Create S3 Buckets

Create two private S3 buckets for hosting static files:

```bash
# For app frontend
aws s3 mb s3://yourorg-prod-app --region us-east-2

# For admin frontend
aws s3 mb s3://yourorg-prod-admin --region us-east-2
```

**Important Configuration**:
- Keep buckets private (block all public access)
- CloudFront will access via Origin Access Identity (OAI)
- No static website hosting needed

### 2. Create CloudFront Origin Access Identities

Create OAIs to allow CloudFront to access private S3 buckets:

```bash
# For app frontend
aws cloudfront create-cloud-front-origin-access-identity \
  --cloud-front-origin-access-identity-config \
  CallerReference=app-$(date +%s),Comment="App frontend OAI"

# For admin frontend
aws cloudfront create-cloud-front-origin-access-identity \
  --cloud-front-origin-access-identity-config \
  CallerReference=admin-$(date +%s),Comment="Admin frontend OAI"
```

**Update S3 Bucket Policies**:

For each bucket, add a policy to allow the OAI to read objects:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "AWS": "arn:aws:iam::cloudfront:user/CloudFront Origin Access Identity YOUR_OAI_ID"
    },
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::yourorg-prod-app/*"
  }]
}
```

Replace `YOUR_OAI_ID` with the ID from the create command output.

### 3. Request SSL Certificates

**Important**: Certificates for CloudFront must be created in the `us-east-1` region.

```bash
# For app.yourdomain.com
aws acm request-certificate \
  --domain-name app.yourdomain.com \
  --validation-method DNS \
  --region us-east-1

# For admin.yourdomain.com
aws acm request-certificate \
  --domain-name admin.yourdomain.com \
  --validation-method DNS \
  --region us-east-1
```

**Validate Certificates**:
1. ACM will provide CNAME records for validation
2. Add these CNAME records to your DNS
3. Wait for certificates to be issued (usually < 30 minutes)

### 4. Create CloudFront Function for SPA Routing

Create a function to handle client-side routing (redirect to index.html):

**File: `spa-router.js`**
```javascript
function handler(event) {
    var request = event.request;
    var uri = request.uri;

    // If URI doesn't have a file extension, route to index.html
    // This handles client-side routing for React apps
    if (!uri.includes('.')) {
        request.uri = '/index.html';
    }

    return request;
}
```

**Create and publish the function**:

```bash
# Create the function
aws cloudfront create-function \
  --name spa-router \
  --function-config Comment="SPA routing for React apps",Runtime="cloudfront-js-1.0" \
  --function-code fileb://spa-router.js

# Publish it (use the ETag from previous command output)
aws cloudfront publish-function \
  --name spa-router \
  --if-match ETAG_FROM_CREATE_OUTPUT
```

### 5. Create CloudFront Distributions

Create two distributions, one for each frontend.

**App Distribution Configuration** (`app-distribution.json`):

```json
{
  "CallerReference": "app-frontend-2024",
  "Comment": "App frontend distribution",
  "Enabled": true,
  "Origins": {
    "Quantity": 1,
    "Items": [{
      "Id": "S3-yourorg-prod-app",
      "DomainName": "yourorg-prod-app.s3.us-east-2.amazonaws.com",
      "S3OriginConfig": {
        "OriginAccessIdentity": "origin-access-identity/cloudfront/YOUR_OAI_ID"
      }
    }]
  },
  "DefaultRootObject": "index.html",
  "DefaultCacheBehavior": {
    "TargetOriginId": "S3-yourorg-prod-app",
    "ViewerProtocolPolicy": "redirect-to-https",
    "AllowedMethods": {
      "Quantity": 3,
      "Items": ["GET", "HEAD", "OPTIONS"],
      "CachedMethods": {
        "Quantity": 3,
        "Items": ["GET", "HEAD", "OPTIONS"]
      }
    },
    "Compress": true,
    "ForwardedValues": {
      "QueryString": false,
      "Cookies": {"Forward": "none"}
    },
    "MinTTL": 0,
    "DefaultTTL": 86400,
    "MaxTTL": 31536000,
    "FunctionAssociations": {
      "Quantity": 1,
      "Items": [{
        "FunctionARN": "arn:aws:cloudfront::ACCOUNT_ID:function/spa-router",
        "EventType": "viewer-request"
      }]
    }
  },
  "CustomErrorResponses": {
    "Quantity": 2,
    "Items": [
      {
        "ErrorCode": 404,
        "ResponseCode": "200",
        "ResponsePagePath": "/index.html",
        "ErrorCachingMinTTL": 0
      },
      {
        "ErrorCode": 403,
        "ResponseCode": "200",
        "ResponsePagePath": "/index.html",
        "ErrorCachingMinTTL": 0
      }
    ]
  },
  "Aliases": {
    "Quantity": 1,
    "Items": ["app.yourdomain.com"]
  },
  "ViewerCertificate": {
    "ACMCertificateArn": "arn:aws:acm:us-east-1:ACCOUNT_ID:certificate/CERT_ID",
    "SSLSupportMethod": "sni-only",
    "MinimumProtocolVersion": "TLSv1.2_2021"
  },
  "PriceClass": "PriceClass_100",
  "HttpVersion": "http2"
}
```

**Create the distributions**:

```bash
aws cloudfront create-distribution \
  --distribution-config file://app-distribution.json

aws cloudfront create-distribution \
  --distribution-config file://admin-distribution.json
```

Note the distribution IDs from the output - you'll need these for cache invalidation.

### 6. Update IAM Permissions

Add S3 and CloudFront permissions to your GitHub Actions IAM user:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::yourorg-prod-app",
        "arn:aws:s3:::yourorg-prod-app/*",
        "arn:aws:s3:::yourorg-prod-admin",
        "arn:aws:s3:::yourorg-prod-admin/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": "cloudfront:CreateInvalidation",
      "Resource": [
        "arn:aws:cloudfront::ACCOUNT_ID:distribution/YOUR_APP_DIST_ID",
        "arn:aws:cloudfront::ACCOUNT_ID:distribution/YOUR_ADMIN_DIST_ID"
      ]
    }
  ]
}
```

### 7. Create GitHub Actions Workflow

Create `.github/workflows/deploy-frontend-prod.yaml`:

```yaml
name: Deploy Frontend to Production
on:
  push:
    branches:
      - main
    paths:
      - 'app/**'
      - 'admin/**'
      - '.github/workflows/deploy-frontend-prod.yaml'

env:
  AWS_REGION: us-east-2
  APP_DISTRIBUTION_ID: YOUR_APP_CLOUDFRONT_DISTRIBUTION_ID
  ADMIN_DISTRIBUTION_ID: YOUR_ADMIN_CLOUDFRONT_DISTRIBUTION_ID

jobs:
  deploy-app:
    name: Deploy App Frontend
    runs-on: ubuntu-latest
    environment: production
    defaults:
      run:
        working-directory: ./app
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'npm'
          cache-dependency-path: ./app/package-lock.json

      - name: Install dependencies
        run: npm ci

      - name: Build
        env:
          APP_API_URL: https://api.yourdomain.com/internal/connect
          APP_APP_URL: https://app.yourdomain.com
          APP_PUBLIC_API_URL: https://api.yourdomain.com
        run: npm run build

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID_PROD }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY_PROD }}
          aws-region: ${{ env.AWS_REGION }}

      - name: Sync static assets to S3 (with long cache)
        run: |
          aws s3 sync public/ s3://yourorg-prod-app/ \
            --delete \
            --cache-control "public,max-age=31536000,immutable" \
            --exclude "*.html" \
            --exclude "config.json"

      - name: Sync HTML files to S3 (with no cache)
        run: |
          aws s3 sync public/ s3://yourorg-prod-app/ \
            --cache-control "public,max-age=0,must-revalidate" \
            --exclude "*" \
            --include "*.html" \
            --include "config.json"

      - name: Invalidate CloudFront cache
        run: |
          aws cloudfront create-invalidation \
            --distribution-id ${{ env.APP_DISTRIBUTION_ID }} \
            --paths "/*"

  deploy-admin:
    name: Deploy Admin Frontend
    runs-on: ubuntu-latest
    environment: production
    defaults:
      run:
        working-directory: ./admin
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'npm'
          cache-dependency-path: ./admin/package-lock.json

      - name: Install dependencies
        run: npm ci

      - name: Build
        env:
          ADMIN_API_URL: https://api.yourdomain.com/internal/connect
        run: npm run build

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID_PROD }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY_PROD }}
          aws-region: ${{ env.AWS_REGION }}

      - name: Sync static assets to S3 (with long cache)
        run: |
          aws s3 sync public/ s3://yourorg-prod-admin/ \
            --delete \
            --cache-control "public,max-age=31536000,immutable" \
            --exclude "*.html" \
            --exclude "config.json"

      - name: Sync HTML files to S3 (with no cache)
        run: |
          aws s3 sync public/ s3://yourorg-prod-admin/ \
            --cache-control "public,max-age=0,must-revalidate" \
            --exclude "*" \
            --include "*.html" \
            --include "config.json"

      - name: Invalidate CloudFront cache
        run: |
          aws cloudfront create-invalidation \
            --distribution-id ${{ env.ADMIN_DISTRIBUTION_ID }} \
            --paths "/*"
```

### 8. Configure DNS

Add CNAME records pointing to your CloudFront distributions:

```
app.yourdomain.com    CNAME  d1234abcd5678.cloudfront.net
admin.yourdomain.com  CNAME  d9876efgh5432.cloudfront.net
```

Get the CloudFront domain names from the AWS Console or CLI:

```bash
aws cloudfront get-distribution --id YOUR_DISTRIBUTION_ID \
  --query 'Distribution.DomainName' --output text
```

## Cache Strategy

The deployment uses two-tier caching:

1. **Static Assets** (JS, CSS, images):
   - Cache-Control: `public,max-age=31536000,immutable`
   - Cached for 1 year (effectively forever)
   - Filenames should include content hashes for cache busting

2. **HTML Files & Config**:
   - Cache-Control: `public,max-age=0,must-revalidate`
   - Always revalidated
   - Ensures users get latest version

After deployment, CloudFront cache is invalidated for immediate updates.

## Cost Estimate

For moderate traffic (assuming ~100GB/month, ~1M requests):

- **S3 Storage**: ~$2/month
- **S3 Requests**: ~$1/month
- **CloudFront Data Transfer**: ~$8-10/month
- **CloudFront Requests**: ~$1/month
- **CloudFront Invalidations**: ~$1/month (1,000 free paths per month)
- **ACM Certificates**: Free
- **Route 53** (if used): $0.50/month per hosted zone

**Total**: ~$13-15/month for both frontends

## Advantages over Netlify

1. **Better AWS Integration**: Everything in one ecosystem
2. **No Build Minute Limits**: Use GitHub Actions' generous limits
3. **Cost Efficiency**: Usually cheaper for moderate traffic
4. **More Control**: Full access to CloudFront settings
5. **Better Performance**: Configure caching exactly as needed

## Disadvantages

1. **More Complex**: More services to configure and maintain
2. **No Preview Deploys**: Would need custom solution (can be built)
3. **No Built-in Forms**: Need to implement separately if needed
4. **Manual Certificate Renewal**: ACM handles this, but need to monitor

## Testing

After deployment, verify:

```bash
# Check HTTPS is working
curl -I https://app.yourdomain.com

# Verify SPA routing (should return 200, not 404)
curl -I https://app.yourdomain.com/some/client/side/route

# Check cache headers
curl -I https://app.yourdomain.com/index.js

# Test from different regions
# CloudFront edge locations should serve cached content
```

## Troubleshooting

### "Access Denied" errors
- Check S3 bucket policy includes the correct OAI
- Verify CloudFront distribution is using the correct OAI

### 404 errors on client-side routes
- Ensure CloudFront Function is attached to viewer-request
- Check custom error responses are configured (403→200, 404→200)

### Stale content after deployment
- Run CloudFront invalidation: `aws cloudfront create-invalidation --distribution-id ID --paths "/*"`
- Check cache-control headers on uploaded files

### Certificate issues
- Ensure certificate is in us-east-1 region
- Verify certificate is validated and issued before attaching to CloudFront
- Check domain name in certificate matches CloudFront alias

## Summary

This AWS-based solution provides the same functionality as Netlify but with more control and better integration with your existing AWS infrastructure. It's well-suited for production deployments where you want full control over the hosting environment.
