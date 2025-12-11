# E2E Tests - Quick Start Guide

## 1-Minute Setup

**Command:**

```bash
# Navigate to project
cd /Users/andrebassi/works/.solutions/operators/infra-operator

# Install dependencies (if not done)
go mod download

# Start LocalStack + Run Tests + Stop LocalStack (all in one)
make test-e2e-localstack
```

That's it! Tests will run automatically.

## Step-by-Step (5 minutes)

### Step 1: Start LocalStack

**Command:**

```bash
make localstack-start
```

Expected output:
```
Creating infra-operator-e2e-localstack ... done
Waiting for LocalStack to be ready...
```

### Step 2: Verify LocalStack is Healthy

**Command:**

```bash
make localstack-health
```

Expected output:
```json
{
  "services": {
"ec2": "running",
"s3": "running",
"rds": "running",
...
  }
}
```

### Step 3: Run All E2E Tests

**Command:**

```bash
make test-e2e
```

Expected output:
```
Running Suite: E2E Suite - /path/to/test/e2e
...
Ran 61 of 61 Specs in 15.234 seconds
SUCCESS! -- 61 Passed | 0 Failed | 0 Pending | 0 Skipped
PASS
```

### Step 4: Stop LocalStack

**Command:**

```bash
make localstack-stop
```

## Run Specific Test Suites

**Command:**

```bash
# VPC tests only (3-4 minutes)
make test-e2e-vpc

# S3 tests only (3-4 minutes)
make test-e2e-s3

# ElasticIP tests only (2-3 minutes)
make test-e2e-elasticip

# Integration tests (5-8 minutes)
make test-e2e-integration

# Drift detection tests (4-5 minutes)
make test-e2e-drift
```

## Run Single Test by Name

**Command:**

```bash
# Focus on specific test
make test-e2e-focus FOCUS="should create a VPC successfully"

# Focus on context
make test-e2e-focus FOCUS="VPC Lifecycle"

# Focus on resource
make test-e2e-focus FOCUS="S3Bucket"
```

## Troubleshooting

### LocalStack Not Starting

**Command:**

```bash
# Check Docker
docker ps

# Restart LocalStack
make localstack-restart

# View logs
make localstack-logs
```

### Tests Failing

**Command:**

```bash
# View LocalStack logs
make localstack-logs

# Check LocalStack health
make localstack-health

# Run with verbose output (already default)
make test-e2e
```

### Clean Up Everything

**Command:**

```bash
# Stop LocalStack
make localstack-stop

# Clean up Docker volumes
docker volume prune -f

# Restart from scratch
make localstack-start
```

## Test Against Real AWS (Advanced)

⚠️ **WARNING: This will create real AWS resources and incur costs!**

**Command:**

```bash
# Export AWS credentials
export AWS_ACCESS_KEY_ID=<your-key>
export AWS_SECRET_ACCESS_KEY=<your-secret>
export AWS_REGION=us-east-1

# Run tests
make test-e2e-real-aws

# Or manually
USE_LOCALSTACK=false make test-e2e
```

## What Gets Tested

### VPC Tests (12 tests)
✅ Create VPC with CIDR
✅ Delete VPC
✅ Update VPC tags
✅ Custom instance tenancy
✅ Deletion policy Retain
✅ Invalid CIDR validation
✅ CIDR size limits (/16 to /28)
✅ LastSyncTime updates
✅ IsDefault flag
✅ Missing provider error
✅ Multiple VPCs

### S3 Tests (10 tests)
✅ Create S3 bucket
✅ Delete S3 bucket
✅ Update bucket tags
✅ Deletion policy Retain
✅ Versioning enabled
✅ Encryption configuration
✅ Lifecycle rules
✅ Public access block
✅ LastSyncTime updates
✅ Multiple buckets

### ElasticIP Tests (10 tests)
✅ Allocate Elastic IP
✅ Release Elastic IP
✅ Update EIP tags
✅ Deletion policy Retain
✅ VPC domain
✅ Standard domain
✅ NetworkBorderGroup
✅ Public IP validation
✅ Invalid domain rejection
✅ Multiple EIPs

### Integration Tests (6 scenarios)
✅ VPC + Subnet + EC2 stack
✅ S3 + IAM role stack
✅ Parallel resource creation
✅ Resource dependencies (VPC + IGW)
✅ Update cascade
✅ Cross-region (skipped)

### Drift Tests (10 scenarios)
✅ VPC tag drift
✅ VPC DNS drift
✅ S3 tag drift
✅ S3 versioning drift
✅ EIP tag drift
✅ EIP association drift
✅ Drift check frequency
✅ Drift severity levels
✅ Multiple resource drift
✅ Drift performance

## Expected Results

**Total Tests:** 61
**Expected Duration:** 15-20 minutes
**Expected Pass Rate:** 100% (LocalStack)

```
Ran 61 of 61 Specs in 15.234 seconds
SUCCESS! -- 61 Passed | 0 Failed | 0 Pending | 0 Skipped
```

## CI/CD

Tests run automatically on GitHub Actions:
- ✅ Pull requests to `main` and `develop`
- ✅ Pushes to `main`
- ✅ Manual workflow dispatch

View workflow: `.github/workflows/e2e.yaml`

## Files Created

**Test Files (8):**
- `test/e2e/suite_test.go`
- `test/e2e/helpers.go`
- `test/e2e/vpc_test.go`
- `test/e2e/s3bucket_test.go`
- `test/e2e/elasticip_test.go`
- `test/e2e/route53_test.go`
- `test/e2e/integration_test.go`
- `test/e2e/drift_test.go`

**Config Files (3):**
- `docker-compose.e2e.yaml`
- `.github/workflows/e2e.yaml`
- `Makefile` (updated)

**Documentation (4):**
- `test/e2e/README.md` - Complete guide
- `E2E_TESTS_SUMMARY.md` - Implementation summary
- `E2E_STATISTICS.md` - Statistics report
- `QUICKSTART_E2E.md` - This file

## Help

View all available commands:
```bash
make help
```

View E2E documentation:
```bash
cat test/e2e/README.md
```

View statistics:
```bash
cat E2E_STATISTICS.md
```

## Support

For issues or questions:
1. Check `test/e2e/README.md`
2. Check LocalStack logs: `make localstack-logs`
3. Check LocalStack health: `make localstack-health`
4. Restart LocalStack: `make localstack-restart`

---

**Happy Testing! 🚀**
