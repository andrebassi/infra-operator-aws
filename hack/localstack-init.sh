#!/bin/bash

# LocalStack initialization script
# This script runs when LocalStack is ready and sets up initial resources for testing

set -e

echo "🚀 Initializing LocalStack for infra-operator testing..."

# AWS CLI endpoint
AWS_ENDPOINT="http://localhost:4566"
AWS_REGION="us-east-1"

# Wait for LocalStack to be fully ready
echo "⏳ Waiting for LocalStack services..."
sleep 5

# Create test S3 buckets
echo "📦 Creating test S3 buckets..."
awslocal s3 mb s3://test-bucket-1 --region $AWS_REGION || true
awslocal s3 mb s3://test-bucket-2 --region $AWS_REGION || true

# Create test SQS queues
echo "📬 Creating test SQS queues..."
awslocal sqs create-queue --queue-name test-queue-1 --region $AWS_REGION || true
awslocal sqs create-queue --queue-name test-queue-2 --region $AWS_REGION || true

# Create test SNS topics
echo "📢 Creating test SNS topics..."
awslocal sns create-topic --name test-topic-1 --region $AWS_REGION || true
awslocal sns create-topic --name test-topic-2 --region $AWS_REGION || true

# Create test DynamoDB table
echo "🗄️  Creating test DynamoDB table..."
awslocal dynamodb create-table \
    --table-name test-table \
    --attribute-definitions AttributeName=id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --region $AWS_REGION || true

# Create test KMS key
echo "🔐 Creating test KMS key..."
awslocal kms create-key \
    --description "Test KMS key for infra-operator" \
    --region $AWS_REGION || true

# Create test Secrets Manager secret
echo "🔑 Creating test Secrets Manager secret..."
awslocal secretsmanager create-secret \
    --name test-secret \
    --secret-string '{"username":"testuser","password":"testpass"}' \
    --region $AWS_REGION || true

# Create test VPC for RDS/EC2 testing
echo "🌐 Creating test VPC..."
VPC_ID=$(awslocal ec2 create-vpc \
    --cidr-block 10.0.0.0/16 \
    --region $AWS_REGION \
    --query 'Vpc.VpcId' \
    --output text 2>/dev/null || echo "")

if [ -n "$VPC_ID" ]; then
    echo "✅ VPC created: $VPC_ID"

    # Create subnet
    SUBNET_ID=$(awslocal ec2 create-subnet \
        --vpc-id $VPC_ID \
        --cidr-block 10.0.1.0/24 \
        --region $AWS_REGION \
        --query 'Subnet.SubnetId' \
        --output text || echo "")

    if [ -n "$SUBNET_ID" ]; then
        echo "✅ Subnet created: $SUBNET_ID"
    fi

    # Create security group
    SG_ID=$(awslocal ec2 create-security-group \
        --group-name test-sg \
        --description "Test security group" \
        --vpc-id $VPC_ID \
        --region $AWS_REGION \
        --query 'GroupId' \
        --output text || echo "")

    if [ -n "$SG_ID" ]; then
        echo "✅ Security group created: $SG_ID"
    fi
fi

echo ""
echo "✅ LocalStack initialization complete!"
echo ""
echo "📊 Resources created:"
echo "  - S3 buckets: test-bucket-1, test-bucket-2"
echo "  - SQS queues: test-queue-1, test-queue-2"
echo "  - SNS topics: test-topic-1, test-topic-2"
echo "  - DynamoDB table: test-table"
echo "  - KMS key: (created)"
echo "  - Secrets Manager: test-secret"
echo "  - VPC: $VPC_ID"
echo "  - Subnet: $SUBNET_ID"
echo "  - Security Group: $SG_ID"
echo ""
echo "🔗 LocalStack endpoint: $AWS_ENDPOINT"
echo "🌍 Region: $AWS_REGION"
echo ""
echo "💡 Test with:"
echo "  awslocal s3 ls"
echo "  awslocal sqs list-queues"
echo "  awslocal dynamodb list-tables"
echo ""
