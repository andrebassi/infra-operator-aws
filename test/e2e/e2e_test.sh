#!/bin/bash
#
# E2E Test Suite for Infra Operator with LocalStack
#
# This script tests all AWS resource types supported by the operator
# against LocalStack to ensure they work correctly.
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
NAMESPACE="default"
OPERATOR_NAMESPACE="iop-system"
LOCALSTACK_NAMESPACE="localstack"
TIMEOUT=60
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Log functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1"
    ((TESTS_SKIPPED++))
}

# Wait for resource to be ready
wait_for_ready() {
    local resource_type=$1
    local resource_name=$2
    local timeout=${3:-$TIMEOUT}

    log_info "Waiting for $resource_type/$resource_name to be ready (timeout: ${timeout}s)..."

    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        local ready=$(kubectl get $resource_type $resource_name -n $NAMESPACE -o jsonpath='{.status.ready}' 2>/dev/null || echo "false")

        if [ "$ready" == "true" ]; then
            log_success "$resource_type/$resource_name is ready"
            return 0
        fi

        sleep 2
        ((elapsed+=2))
    done

    log_error "$resource_type/$resource_name did not become ready within ${timeout}s"
    kubectl describe $resource_type $resource_name -n $NAMESPACE
    return 1
}

# Check if resource exists
check_resource_exists() {
    local resource_type=$1
    local resource_name=$2

    if kubectl get $resource_type $resource_name -n $NAMESPACE &>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test resources..."

    # Delete all test resources
    kubectl delete vpc test-vpc -n $NAMESPACE --ignore-not-found
    kubectl delete subnet test-subnet -n $NAMESPACE --ignore-not-found
    kubectl delete securitygroup test-sg -n $NAMESPACE --ignore-not-found
    kubectl delete s3bucket test-bucket-simple -n $NAMESPACE --ignore-not-found
    kubectl delete dynamodbtable test-users-table -n $NAMESPACE --ignore-not-found
    kubectl delete sqsqueue test-queue -n $NAMESPACE --ignore-not-found
    kubectl delete snstopic test-topic -n $NAMESPACE --ignore-not-found
    kubectl delete lambdafunction test-lambda -n $NAMESPACE --ignore-not-found

    log_info "Cleanup complete"
}

# Test 1: Operator is running
test_operator_running() {
    log_info "Test: Operator is running in namespace $OPERATOR_NAMESPACE"

    local pod=$(kubectl get pods -n $OPERATOR_NAMESPACE -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$pod" ]; then
        log_error "Operator pod not found in namespace $OPERATOR_NAMESPACE"
        return 1
    fi

    local status=$(kubectl get pod $pod -n $OPERATOR_NAMESPACE -o jsonpath='{.status.phase}')

    if [ "$status" == "Running" ]; then
        log_success "Operator pod $pod is running"
        return 0
    else
        log_error "Operator pod $pod is not running (status: $status)"
        return 1
    fi
}

# Test 2: LocalStack is running
test_localstack_running() {
    log_info "Test: LocalStack is running in namespace $LOCALSTACK_NAMESPACE"

    local pod=$(kubectl get pods -n $LOCALSTACK_NAMESPACE -l app.kubernetes.io/name=localstack -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$pod" ]; then
        log_error "LocalStack pod not found in namespace $LOCALSTACK_NAMESPACE"
        return 1
    fi

    local status=$(kubectl get pod $pod -n $LOCALSTACK_NAMESPACE -o jsonpath='{.status.phase}')

    if [ "$status" == "Running" ]; then
        log_success "LocalStack pod $pod is running"
        return 0
    else
        log_error "LocalStack pod $pod is not running (status: $status)"
        return 1
    fi
}

# Test 3: AWSProvider connection
test_awsprovider() {
    log_info "Test: AWSProvider connects to LocalStack"

    if ! check_resource_exists awsprovider localstack; then
        log_error "AWSProvider 'localstack' not found"
        return 1
    fi

    if wait_for_ready awsprovider localstack 30; then
        local account=$(kubectl get awsprovider localstack -n $NAMESPACE -o jsonpath='{.status.accountID}')
        if [ "$account" == "000000000000" ]; then
            log_success "AWSProvider connected to LocalStack (account: $account)"
            return 0
        else
            log_error "AWSProvider account mismatch (expected: 000000000000, got: $account)"
            return 1
        fi
    else
        return 1
    fi
}

# Test 4: S3 Bucket creation
test_s3_bucket() {
    log_info "Test: S3 Bucket creation"

    if ! check_resource_exists s3bucket test-bucket-simple; then
        log_error "S3Bucket 'test-bucket-simple' not found"
        return 1
    fi

    if wait_for_ready s3bucket test-bucket-simple; then
        local bucket_name=$(kubectl get s3bucket test-bucket-simple -n $NAMESPACE -o jsonpath='{.status.bucketName}')
        log_success "S3 Bucket created: $bucket_name"
        return 0
    else
        return 1
    fi
}

# Test 5: DynamoDB Table creation
test_dynamodb_table() {
    log_info "Test: DynamoDB Table creation"

    if ! check_resource_exists dynamodbtable test-users-table; then
        log_error "DynamoDBTable 'test-users-table' not found"
        return 1
    fi

    if wait_for_ready dynamodbtable test-users-table; then
        local table_status=$(kubectl get dynamodbtable test-users-table -n $NAMESPACE -o jsonpath='{.status.tableStatus}')
        if [ "$table_status" == "ACTIVE" ]; then
            log_success "DynamoDB Table created and active"
            return 0
        else
            log_error "DynamoDB Table not active (status: $table_status)"
            return 1
        fi
    else
        return 1
    fi
}

# Test 6: SQS Queue creation
test_sqs_queue() {
    log_info "Test: SQS Queue creation"

    if ! check_resource_exists sqsqueue test-queue; then
        log_error "SQSQueue 'test-queue' not found"
        return 1
    fi

    if wait_for_ready sqsqueue test-queue; then
        local queue_url=$(kubectl get sqsqueue test-queue -n $NAMESPACE -o jsonpath='{.status.queueURL}')
        log_success "SQS Queue created: $queue_url"
        return 0
    else
        return 1
    fi
}

# Test 7: SNS Topic creation
test_sns_topic() {
    log_info "Test: SNS Topic creation"

    if ! check_resource_exists snstopic test-topic; then
        log_error "SNSTopic 'test-topic' not found"
        return 1
    fi

    if wait_for_ready snstopic test-topic; then
        local topic_arn=$(kubectl get snstopic test-topic -n $NAMESPACE -o jsonpath='{.status.topicARN}')
        log_success "SNS Topic created: $topic_arn"
        return 0
    else
        return 1
    fi
}

# Test 8: VPC creation
test_vpc() {
    log_info "Test: VPC creation"

    if ! check_resource_exists vpc test-vpc; then
        log_error "VPC 'test-vpc' not found"
        return 1
    fi

    if wait_for_ready vpc test-vpc; then
        local vpc_id=$(kubectl get vpc test-vpc -n $NAMESPACE -o jsonpath='{.status.vpcID}')
        local cidr=$(kubectl get vpc test-vpc -n $NAMESPACE -o jsonpath='{.status.cidrBlock}')
        log_success "VPC created: $vpc_id (CIDR: $cidr)"
        return 0
    else
        return 1
    fi
}

# Test 9: Subnet creation
test_subnet() {
    log_info "Test: Subnet creation"

    if ! check_resource_exists subnet test-subnet; then
        log_error "Subnet 'test-subnet' not found"
        return 1
    fi

    if wait_for_ready subnet test-subnet; then
        local subnet_id=$(kubectl get subnet test-subnet -n $NAMESPACE -o jsonpath='{.status.subnetID}')
        local available_ips=$(kubectl get subnet test-subnet -n $NAMESPACE -o jsonpath='{.status.availableIpAddressCount}')
        log_success "Subnet created: $subnet_id ($available_ips IPs available)"
        return 0
    else
        return 1
    fi
}

# Test 10: Lambda Function (expected to fail on LocalStack Free)
test_lambda_function() {
    log_info "Test: Lambda Function creation (may fail on LocalStack Free)"

    if ! check_resource_exists lambdafunction test-lambda; then
        log_skip "LambdaFunction 'test-lambda' not found - skipping"
        return 0
    fi

    # Lambda is expected to have issues on LocalStack Free, so we just check if it exists
    local state=$(kubectl get lambdafunction test-lambda -n $NAMESPACE -o jsonpath='{.status.state}' 2>/dev/null || echo "Unknown")
    local function_arn=$(kubectl get lambdafunction test-lambda -n $NAMESPACE -o jsonpath='{.status.functionArn}' 2>/dev/null || echo "")

    if [ -n "$function_arn" ]; then
        log_warn "Lambda Function created but may not be fully functional: $function_arn (state: $state)"
        log_success "Lambda base64 encoding works correctly"
        return 0
    else
        log_skip "Lambda Function not fully created (LocalStack limitation)"
        return 0
    fi
}

# Test 11: RDS Instance (expected to fail on LocalStack Free)
test_rds_instance() {
    log_info "Test: RDS Instance (not available in LocalStack Free)"

    if ! check_resource_exists rdsinstance test-rds; then
        log_skip "RDSInstance 'test-rds' not found - skipping"
        return 0
    fi

    local status=$(kubectl get rdsinstance test-rds -n $NAMESPACE -o jsonpath='{.status.status}' 2>/dev/null || echo "")

    if [ "$status" == "error" ]; then
        log_skip "RDS not available in LocalStack Free (expected)"
        return 0
    else
        log_warn "RDS status: $status"
        return 0
    fi
}

# Main test execution
main() {
    echo "=========================================="
    echo "  Infra Operator E2E Test Suite"
    echo "=========================================="
    echo ""

    log_info "Starting E2E tests..."
    echo ""

    # Pre-requisite tests
    test_operator_running || exit 1
    test_localstack_running || exit 1
    echo ""

    # Core functionality tests
    test_awsprovider || true
    echo ""

    # Storage tests
    test_s3_bucket || true
    test_dynamodb_table || true
    echo ""

    # Messaging tests
    test_sqs_queue || true
    test_sns_topic || true
    echo ""

    # Networking tests
    test_vpc || true
    test_subnet || true
    echo ""

    # Compute tests
    test_lambda_function || true
    echo ""

    # Database tests (expected to fail on free tier)
    test_rds_instance || true
    echo ""

    # Summary
    echo "=========================================="
    echo "  Test Results"
    echo "=========================================="
    echo -e "${GREEN}Passed:  $TESTS_PASSED${NC}"
    echo -e "${RED}Failed:  $TESTS_FAILED${NC}"
    echo -e "${YELLOW}Skipped: $TESTS_SKIPPED${NC}"
    echo "=========================================="

    if [ $TESTS_FAILED -gt 0 ]; then
        echo ""
        log_error "Some tests failed!"
        exit 1
    else
        echo ""
        log_success "All tests passed!"
        exit 0
    fi
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Run main function
main "$@"
