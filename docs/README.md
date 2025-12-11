# Infra Operator Documentation

This directory contains the documentation for the Infra Operator, ready to be deployed with [Mintlify](https://mintlify.com/).

## 📚 Documentation Structure

```
docs/
├── README.md                    # This file
├── mint.json                    # Mintlify configuration
├── SERVICES_GUIDE.md           # Complete services guide (all-in-one)
├── introduction.mdx             # Getting started
├── quickstart.mdx               # Quick start guide
├── installation.mdx             # Installation instructions
├── architecture.mdx             # Architecture overview
├── concepts/                    # Core concepts
│   ├── aws-provider.mdx
│   ├── deletion-policies.mdx
│   ├── clean-architecture.mdx
│   └── troubleshooting.mdx
├── services/                    # Service-specific docs
│   ├── networking/
│   │   ├── vpc.mdx
│   │   ├── subnet.mdx
│   │   ├── internet-gateway.mdx
│   │   ├── nat-gateway.mdx
│   │   ├── route-table.mdx
│   │   └── security-group.mdx
│   ├── storage/
│   │   └── s3.mdx
│   ├── database/
│   │   ├── dynamodb.mdx
│   │   └── rds.mdx
│   ├── compute/
│   │   ├── ec2.mdx
│   │   └── lambda.mdx
│   ├── messaging/
│   │   ├── sqs.mdx
│   │   └── sns.mdx
│   ├── security/
│   │   ├── iam.mdx
│   │   ├── secrets-manager.mdx
│   │   └── kms.mdx
│   ├── container/
│   │   └── ecr.mdx
│   └── caching/
│       └── elasticache.mdx
├── guides/                      # How-to guides
│   ├── multi-tier-network.mdx
│   ├── serverless-app.mdx
│   ├── eks-ready-network.mdx
│   └── best-practices.mdx
└── api-reference/              # API reference
    ├── introduction.mdx
    ├── awsprovider.mdx
    ├── vpc.mdx
    ├── subnet.mdx
    ├── internetgateway.mdx
    ├── natgateway.mdx
    ├── s3bucket.mdx
    ├── dynamodbtable.mdx
    ├── rdsinstance.mdx
    ├── ec2instance.mdx
    ├── lambdafunction.mdx
    ├── sqsqueue.mdx
    ├── snstopic.mdx
    ├── iamrole.mdx
    ├── secretsmanagersecret.mdx
    ├── kmskey.mdx
    ├── ecrrepository.mdx
    └── elasticachecluster.mdx
```

## 🚀 Quick Start with Mintlify

### Prerequisites

- Node.js 18+ installed
- npm or yarn package manager

### Install Mintlify CLI

```bash
npm i -g mintlify
```

### Preview Documentation Locally

```bash
# Navigate to docs directory
cd docs/

# Start Mintlify dev server
mintlify dev
```

The documentation will be available at `http://localhost:3000`

### Build Documentation

```bash
mintlify build
```

## 📝 Creating Documentation Pages

### File Format

All documentation pages use `.mdx` format (Markdown + JSX). This allows you to:
- Use standard Markdown syntax
- Embed React components
- Use Mintlify components (Tabs, CodeGroup, Accordion, etc.)

### Example Page Structure

```mdx
---
title: 'VPC - Virtual Private Cloud'
description: 'Create isolated virtual networks in AWS'
icon: 'network-wired'
---

# VPC - Virtual Private Cloud

Create isolated virtual networks in AWS.

## Overview

Brief description of the service...

## Quick Example

<CodeGroup>
```yaml Basic VPC
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
\```
</CodeGroup>

## Configuration Options

### Required Fields

<ParamField path="cidrBlock" type="string" required>
  The IPv4 CIDR block for the VPC
</ParamField>

### Optional Fields

<ParamField path="enableDnsSupport" type="boolean" default="true">
  Enable DNS resolution in the VPC
</ParamField>

## Status Fields

The VPC status includes:

- `vpcID`: AWS VPC identifier
- `state`: Current state (available, pending, etc.)
- `ready`: Boolean indicating if VPC is ready

## Examples

### Multi-AZ VPC

\```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: production-vpc
    Environment: production
\```

## Troubleshooting

<Accordion title="VPC stuck in pending state">
  Check the controller logs...
</Accordion>

## Next Steps

<CardGroup cols={2}>
  <Card title="Create Subnets" icon="sitemap" href="/services/networking/subnet">
    Learn how to create subnets within your VPC
  </Card>
  <Card title="Internet Gateway" icon="globe" href="/services/networking/internet-gateway">
    Add internet connectivity to your VPC
  </Card>
</CardGroup>
```

## 🎨 Mintlify Components

### CodeGroup (Multiple code examples with tabs)

```mdx
<CodeGroup>
\```yaml Production
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
spec:
  cidrBlock: "10.0.0.0/16"
\```

\```yaml Development
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: dev-vpc
spec:
  cidrBlock: "172.16.0.0/16"
\```
</CodeGroup>
```

### Tabs

```mdx
<Tabs>
  <Tab title="IRSA Authentication">
    Use IAM Roles for Service Accounts...
  </Tab>
  <Tab title="Static Credentials">
    Use access key and secret...
  </Tab>
</Tabs>
```

### Accordion

```mdx
<Accordion title="Advanced Configuration">
  Details about advanced options...
</Accordion>
```

### Cards

```mdx
<CardGroup cols={2}>
  <Card title="VPC" icon="network-wired" href="/services/networking/vpc">
    Create virtual private clouds
  </Card>
  <Card title="Subnet" icon="sitemap" href="/services/networking/subnet">
    Segment your network
  </Card>
</CardGroup>
```

### Callouts

```mdx
<Note>
  This is a note callout
</Note>

<Warning>
  This is a warning callout
</Warning>

<Tip>
  This is a tip callout
</Tip>

<Info>
  This is an info callout
</Info>
```

### Parameter Fields

```mdx
<ParamField path="cidrBlock" type="string" required>
  The IPv4 CIDR block for the VPC
</ParamField>

<ParamField path="enableDnsSupport" type="boolean" default="true">
  Enable DNS resolution
</ParamField>
```

## 🎯 Content Guidelines

### 1. Service Pages Structure

Each service page should include:
1. **Overview**: Brief description of the service
2. **Quick Example**: Minimal working example
3. **Configuration**: All spec fields documented
4. **Status Fields**: What status information is available
5. **Examples**: Common use cases
6. **Troubleshooting**: Common issues and solutions
7. **Next Steps**: Related services/guides

### 2. Code Examples

- Always use complete, working examples
- Include comments for complex configurations
- Show both simple and advanced examples
- Use realistic values (not `xxx` or `todo`)

### 3. Writing Style

- Use active voice ("Create a VPC" not "A VPC can be created")
- Be concise and clear
- Include "why" not just "what"
- Link to related documentation

### 4. Icons

Use Font Awesome icons in frontmatter:
- `network-wired` for networking
- `database` for databases
- `server` for compute
- `lock` for security
- `envelope` for messaging
- `box` for storage

## 📦 Deployment

### Deploy to Mintlify Cloud

1. **Connect Repository**
   - Go to [Mintlify Dashboard](https://dashboard.mintlify.com)
   - Connect your GitHub repository
   - Point to the `docs/` directory

2. **Configure Domain** (optional)
   - Add custom domain in Mintlify dashboard
   - Update DNS settings

3. **Auto-Deploy**
   - Every push to main branch auto-deploys
   - Preview deployments for PRs

### Deploy to Custom Hosting

```bash
# Build static site
mintlify build

# Output will be in .mintlify/
# Deploy .mintlify/ to your hosting (Vercel, Netlify, etc.)
```

## 🔧 Customization

### Colors

Edit `mint.json`:
```json
{
  "colors": {
    "primary": "#FF6B35",
    "light": "#FF8C61",
    "dark": "#FF4500"
  }
}
```

### Logo

Add your logos to:
- `docs/logo/light.svg` (for light mode)
- `docs/logo/dark.svg` (for dark mode)

### Favicon

Add `docs/favicon.svg`

### Navigation

Edit `navigation` array in `mint.json`:
```json
{
  "navigation": [
    {
      "group": "Get Started",
      "pages": ["introduction", "quickstart"]
    }
  ]
}
```

## 📚 Resources

- [Mintlify Documentation](https://mintlify.com/docs)
- [Mintlify Components](https://mintlify.com/docs/components)
- [Mintlify Examples](https://github.com/mintlify/starter)

## 🤝 Contributing

When adding new services:

1. Create service documentation in `services/<category>/<service>.mdx`
2. Add API reference in `api-reference/<service>.mdx`
3. Update `mint.json` navigation
4. Add examples to guides if applicable
5. Update SERVICES_GUIDE.md with new service info

## 📝 TODO

- [ ] Create individual .mdx files for each service
- [ ] Add screenshots and diagrams
- [ ] Create video tutorials
- [ ] Add interactive examples
- [ ] Write migration guides from other tools
- [ ] Add Terraform comparison guide
- [ ] Create troubleshooting flowcharts

## 🎓 Next Steps

1. **Preview locally**: `mintlify dev`
2. **Create service pages**: Start with networking services
3. **Add examples**: Real-world use cases
4. **Deploy**: Push to Mintlify cloud or custom hosting

---

**Generated by Infra Operator**
Version: v1.0.0
Last Updated: 2025-11-22
