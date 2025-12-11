#!/usr/bin/env bash
#
# Test Helm chart installation in a test namespace
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CHART_DIR="${REPO_ROOT}/chart"
TEST_NAMESPACE="infra-operator-test"
RELEASE_NAME="infra-operator-test"

echo "========================================="
echo "Infra Operator - Helm Chart Testing"
echo "========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Validate dependencies
command -v helm >/dev/null 2>&1 || { echo -e "${RED}Error: helm is not installed${NC}" >&2; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo -e "${RED}Error: kubectl is not installed${NC}" >&2; exit 1; }

# Check if kubectl can connect to cluster
if ! kubectl cluster-info > /dev/null 2>&1; then
    echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
    exit 1
fi

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up test resources...${NC}"
    helm uninstall "${RELEASE_NAME}" -n "${TEST_NAMESPACE}" 2>/dev/null || true
    kubectl delete namespace "${TEST_NAMESPACE}" 2>/dev/null || true
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

trap cleanup EXIT

# Parse arguments
VALUES_FILE="${1:-values-dev.yaml}"
if [[ ! -f "${CHART_DIR}/${VALUES_FILE}" ]]; then
    echo -e "${RED}Error: Values file not found: ${VALUES_FILE}${NC}"
    exit 1
fi

echo -e "${BLUE}Configuration:${NC}"
echo "  Chart: ${CHART_DIR}"
echo "  Values: ${VALUES_FILE}"
echo "  Namespace: ${TEST_NAMESPACE}"
echo "  Release: ${RELEASE_NAME}"
echo ""

# Create test namespace
echo -e "${YELLOW}[1/7] Creating test namespace...${NC}"
kubectl create namespace "${TEST_NAMESPACE}" 2>/dev/null || true
echo -e "${GREEN}✓ Namespace ready${NC}"
echo ""

# Lint chart
echo -e "${YELLOW}[2/7] Linting chart...${NC}"
if helm lint "${CHART_DIR}" --values "${CHART_DIR}/${VALUES_FILE}"; then
    echo -e "${GREEN}✓ Chart linting passed${NC}"
else
    echo -e "${RED}✗ Chart linting failed${NC}"
    exit 1
fi
echo ""

# Dry-run install
echo -e "${YELLOW}[3/7] Running dry-run installation...${NC}"
if helm install "${RELEASE_NAME}" "${CHART_DIR}" \
    --namespace "${TEST_NAMESPACE}" \
    --values "${CHART_DIR}/${VALUES_FILE}" \
    --dry-run --debug > /tmp/helm-dry-run.yaml 2>&1; then
    echo -e "${GREEN}✓ Dry-run successful${NC}"
else
    echo -e "${RED}✗ Dry-run failed${NC}"
    cat /tmp/helm-dry-run.yaml
    exit 1
fi
echo ""

# Install chart
echo -e "${YELLOW}[4/7] Installing chart...${NC}"
if helm install "${RELEASE_NAME}" "${CHART_DIR}" \
    --namespace "${TEST_NAMESPACE}" \
    --values "${CHART_DIR}/${VALUES_FILE}" \
    --wait --timeout 5m; then
    echo -e "${GREEN}✓ Chart installed${NC}"
else
    echo -e "${RED}✗ Chart installation failed${NC}"
    kubectl get all -n "${TEST_NAMESPACE}"
    kubectl logs -n "${TEST_NAMESPACE}" -l app.kubernetes.io/name=infra-operator --tail=50
    exit 1
fi
echo ""

# Check deployment status
echo -e "${YELLOW}[5/7] Checking deployment status...${NC}"
if kubectl rollout status deployment -n "${TEST_NAMESPACE}" -l app.kubernetes.io/name=infra-operator --timeout=2m; then
    echo -e "${GREEN}✓ Deployment ready${NC}"
else
    echo -e "${RED}✗ Deployment not ready${NC}"
    kubectl describe deployment -n "${TEST_NAMESPACE}" -l app.kubernetes.io/name=infra-operator
    exit 1
fi
echo ""

# Run Helm tests
echo -e "${YELLOW}[6/7] Running Helm tests...${NC}"
if helm test "${RELEASE_NAME}" -n "${TEST_NAMESPACE}" --timeout 2m; then
    echo -e "${GREEN}✓ Helm tests passed${NC}"
else
    echo -e "${RED}✗ Helm tests failed${NC}"
    kubectl logs -n "${TEST_NAMESPACE}" -l helm.sh/test=true
    exit 1
fi
echo ""

# Show release info
echo -e "${YELLOW}[7/7] Gathering release information...${NC}"
echo ""
echo -e "${BLUE}Release Status:${NC}"
helm status "${RELEASE_NAME}" -n "${TEST_NAMESPACE}"
echo ""
echo -e "${BLUE}Deployed Resources:${NC}"
kubectl get all -n "${TEST_NAMESPACE}"
echo ""
echo -e "${BLUE}CRDs Installed:${NC}"
kubectl get crds | grep infra.io || echo "No CRDs found"
echo ""

# Success
echo "========================================="
echo -e "${GREEN}All Tests Passed!${NC}"
echo "========================================="
echo ""
echo "The chart has been successfully tested."
echo "Test namespace ${TEST_NAMESPACE} will be cleaned up on exit."
echo ""
echo "To keep the test deployment, press Ctrl+C now."
echo "Otherwise, cleanup will happen in 10 seconds..."
sleep 10
