---
title: 'Route Table - Routing Table'
description: 'Configure traffic routing in the VPC'
sidebar_position: 6
---

Configure route tables to control network traffic flow in VPC subnets.

## Prerequisites

Before creating a Route Table, you need:

1. **AWSProvider** configured
2. **VPC** created
3. **Gateways** (Internet Gateway, NAT Gateway, etc.) if needed

## Overview

A Route Table contains a set of rules (routes) that determine where network traffic from the subnet is directed. Each subnet must be associated with a route table.

**Main components:**
- 🛣️ **Routes** - Traffic routing rules
- 🔗 **Associations** - Link with subnets
- 🎯 **Destinations** - Internet Gateway, NAT Gateway, VPC Peering, etc.

## Quick Start

**Public Route Table (Internet):**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: public-route-table
  namespace: default
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-abc123
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      gatewayID: igw-abc123  # Internet Gateway
  subnetAssociations:
- subnet-public-1a
- subnet-public-1b
  tags:
    Name: public-route-table
    Type: public
  deletionPolicy: Delete
```

**Private Route Table (NAT):**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: private-route-table-1a
  namespace: default
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-abc123
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      natGatewayID: nat-abc123  # NAT Gateway
  subnetAssociations:
- subnet-private-1a
  tags:
    Name: private-route-table-1a
    Type: private
    AZ: us-east-1a
  deletionPolicy: Delete
```

**Route Table with VPC Peering:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: peering-route-table
  namespace: default
spec:
  providerRef:
    name: production-aws
  vpcID: vpc-abc123
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      gatewayID: igw-abc123
- destinationCidrBlock: "10.1.0.0/16"
      vpcPeeringConnectionID: pcx-abc123  # VPC Peering
  subnetAssociations:
- subnet-app-1a
- subnet-app-1b
  tags:
    Name: peering-route-table
    Peering: vpc-staging
  deletionPolicy: Delete
```

**Apply:**

```bash
kubectl apply -f route-table.yaml
```

**Verify Status:**

```bash
kubectl get routetable
kubectl describe routetable public-route-table
```
## Configuration Reference

### Required Fields

Reference to the AWSProvider resource

  AWSProvider resource name

VPC ID where the route table will be created

  Example: `vpc-abc123`

### Optional Fields

List of routes to be added

  Traffic destination CIDR

      Examples:
- `0.0.0.0/0` - Default route (all traffic)
- `10.1.0.0/16` - Specific network

Internet Gateway ID

      Use for public routes. Example: `igw-abc123`

NAT Gateway ID

      Use for private routes with internet access. Example: `nat-abc123`

VPC Peering connection ID

      Use for routing between VPCs. Example: `pcx-abc123`

Transit Gateway ID

      Use for hub-and-spoke architectures. Example: `tgw-abc123`

:::note

**Only one** of the destination fields (gatewayID, natGatewayID, vpcPeeringConnectionID, transitGatewayID) must be specified per route.

:::


List of subnet IDs to associate with this route table

  Example:
  ```yaml
  subnetAssociations:
- subnet-abc123
- subnet-def456
  ```

Custom tags for the route table

  Example:
  ```yaml
  tags:
    Name: my-route-table
    Environment: production
    Type: public
  ```

Resource deletion policy

  **Possible values:**
  - `Delete` - Delete route table when removing CR
  - `Retain` - Keep route table when removing CR

## Status

The route table status is automatically updated:

```yaml
status:
  ready: true
  routeTableID: rtb-abc123
  vpcID: vpc-abc123
  associationIDs:
- rtbassoc-abc123
- rtbassoc-def456
  lastSyncTime: "2025-01-23T10:30:00Z"
```

### Status Fields

Indicates if the route table is ready

Route table ID created in AWS

Subnet association IDs

## Use Cases

### 1. Multi-Tier Architecture (Public + Private)

**Example:**

```yaml
# Public Route Table
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: public-rt
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      gatewayID: igw-prod
  subnetAssociations:
- subnet-public-1a
- subnet-public-1b
- subnet-public-1c
  tags:
    Name: public-route-table
    Tier: public

# Private Route Table - AZ 1a
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: private-rt-1a
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      natGatewayID: nat-1a
  subnetAssociations:
- subnet-private-1a
  tags:
    Name: private-route-table-1a
    Tier: private
    AZ: us-east-1a

# Private Route Table - AZ 1b
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: private-rt-1b
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      natGatewayID: nat-1b
  subnetAssociations:
- subnet-private-1b
  tags:
    Name: private-route-table-1b
    Tier: private
    AZ: us-east-1b
```

### 2. VPC Peering between Environments

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: prod-to-staging-rt
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod
  routes:
# Default route to internet
- destinationCidrBlock: "0.0.0.0/0"
      gatewayID: igw-prod
# Route to staging VPC
- destinationCidrBlock: "10.1.0.0/16"
      vpcPeeringConnectionID: pcx-prod-staging
# Route to shared services VPC
- destinationCidrBlock: "10.2.0.0/16"
      vpcPeeringConnectionID: pcx-prod-shared
  subnetAssociations:
- subnet-app-1a
- subnet-app-1b
  tags:
    Name: prod-peering-route-table
    Environment: production
```

### 3. Isolated Route Table (No Internet)

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: isolated-rt
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod
  # No routes = VPC local traffic only
  routes: []
  subnetAssociations:
- subnet-db-1a
- subnet-db-1b
  tags:
    Name: isolated-route-table
    Tier: database
    Internet: "false"
```

### 4. Transit Gateway Hub

**Example:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: tgw-spoke-rt
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-spoke
  routes:
# Internet via NAT in hub
- destinationCidrBlock: "0.0.0.0/0"
      transitGatewayID: tgw-abc123
# Other VPCs via TGW
- destinationCidrBlock: "10.0.0.0/8"
      transitGatewayID: tgw-abc123
  subnetAssociations:
- subnet-app-1a
- subnet-app-1b
  tags:
    Name: transit-gateway-route-table
    Architecture: hub-spoke
```

## Routing Patterns

### Public Subnet

A public subnet has a `0.0.0.0/0` route to an **Internet Gateway**:

```yaml
routes:
  - destinationCidrBlock: "0.0.0.0/0"
gatewayID: igw-abc123
```

### Private Subnet with Internet

A private subnet with internet access uses a **NAT Gateway**:

```yaml
routes:
  - destinationCidrBlock: "0.0.0.0/0"
natGatewayID: nat-abc123
```

### Isolated Private Subnet

An isolated subnet **does not have** a route to `0.0.0.0/0`:

```yaml
routes: []  # VPC local traffic only
```

## Common Operations

### Add New Route

**Command:**

```bash
kubectl edit routetable my-route-table

# Add to spec.routes:
# - destinationCidrBlock: "10.5.0.0/16"
#   vpcPeeringConnectionID: pcx-new
```

### Associate New Subnet

**Command:**

```bash
kubectl patch routetable my-route-table \
  --type merge \
  -p '{"spec":{"subnetAssociations":["subnet-abc123","subnet-def456","subnet-new789"]}}'
```

### Verify Routes

**Command:**

```bash
# View all routes
kubectl get routetable my-route-table -o jsonpath='{.spec.routes}'

# View associations
kubectl get routetable my-route-table -o jsonpath='{.spec.subnetAssociations}'
```

### Migrate Subnet to Another Route Table

**Command:**

```bash
# 1. Remove from old route table
kubectl patch routetable old-rt \
  --type json \
  -p '[{"op": "remove", "path": "/spec/subnetAssociations/2"}]'

# 2. Add to new route table
kubectl patch routetable new-rt \
  --type merge \
  -p '{"spec":{"subnetAssociations":["subnet-xyz789"]}}'
```

## Troubleshooting

### Route Table does not create routes

### Verify Gateways

**Command:**

```bash
# Verify if the Internet Gateway exists
aws ec2 describe-internet-gateways --internet-gateway-ids igw-abc123

# Verify if the NAT Gateway exists and is available
aws ec2 describe-nat-gateways --nat-gateway-ids nat-abc123
```

### Verify VPC Peering

**Command:**

```bash
# Verify VPC Peering status
aws ec2 describe-vpc-peering-connections \
      --vpc-peering-connection-ids pcx-abc123
```

### View Operator Logs

**Command:**

```bash
kubectl logs -n infra-operator-system \
      -l control-plane=controller-manager \
      | grep RouteTable
```

### Subnet does not appear in associations

**Command:**

```bash
# Verify if the subnet exists
kubectl get subnet my-subnet

# Verify RouteTable status
kubectl describe routetable my-route-table

# View events
kubectl get events --field-selector involvedObject.name=my-route-table
```

### Route conflicts

If there is a conflict error:

```bash
# List VPC route tables
aws ec2 describe-route-tables \
  --filters "Name=vpc-id,Values=vpc-abc123"

# Verify existing routes
aws ec2 describe-route-tables \
  --route-table-ids rtb-abc123 \
  --query 'RouteTables[*].Routes'
```

## Best Practices

:::note Best Practices

- **Separate route tables per tier** — Public, private, and database subnets should have different routing
- **Use 0.0.0.0/0 to IGW for public** — Public subnets route to Internet Gateway
- **Use 0.0.0.0/0 to NAT for private** — Private subnets route through NAT Gateway
- **Tag route tables clearly** — Include subnet-type, environment, AZ in tags
- **Document custom routes** — Keep track of VPC peering, VPN, Direct Connect routes

:::

## Complete Example: Multi-Tier VPC

**Example:**

```yaml
# VPC
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: prod-vpc
spec:
  providerRef:
    name: prod-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true

# Internet Gateway
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
metadata:
  name: prod-igw
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod

# NAT Gateways
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: nat-1a
spec:
  providerRef:
    name: prod-aws
  subnetID: subnet-public-1a

---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
metadata:
  name: nat-1b
spec:
  providerRef:
    name: prod-aws
  subnetID: subnet-public-1b

# Route Tables
---
# Public
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: public-rt
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      gatewayID: igw-prod
  subnetAssociations:
- subnet-public-1a
- subnet-public-1b

---
# Private AZ-1a
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: private-rt-1a
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      natGatewayID: nat-1a
  subnetAssociations:
- subnet-private-1a

---
# Private AZ-1b
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: private-rt-1b
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod
  routes:
- destinationCidrBlock: "0.0.0.0/0"
      natGatewayID: nat-1b
  subnetAssociations:
- subnet-private-1b

---
# Database (isolated)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
metadata:
  name: database-rt
spec:
  providerRef:
    name: prod-aws
  vpcID: vpc-prod
  routes: []  # No internet
  subnetAssociations:
- subnet-db-1a
- subnet-db-1b
```

## Related Resources

- [VPC](/services/networking/vpc)

  - [Subnet](/services/networking/subnet)

  - [Internet Gateway](/services/networking/internet-gateway)

  - [NAT Gateway](/services/networking/nat-gateway)
## References

- [AWS Route Tables Documentation](https://docs.aws.amazon.com/vpc/latest/userguide/VPC_Route_Tables.html)
- [VPC Routing Best Practices](https://docs.aws.amazon.com/vpc/latest/userguide/route-table-options.html)
