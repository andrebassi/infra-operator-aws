#!/usr/bin/env bash
#
# Package Helm chart for distribution
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CHART_DIR="${REPO_ROOT}/chart"
DIST_DIR="${REPO_ROOT}/dist/helm"

echo "========================================="
echo "Infra Operator - Helm Chart Packaging"
echo "========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Validate dependencies
command -v helm >/dev/null 2>&1 || { echo -e "${RED}Error: helm is not installed${NC}" >&2; exit 1; }

# Clean and create dist directory
echo -e "${YELLOW}[1/5] Preparing distribution directory...${NC}"
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"
echo -e "${GREEN}✓ Distribution directory ready${NC}"
echo ""

# Lint chart
echo -e "${YELLOW}[2/5] Linting chart...${NC}"
if helm lint "${CHART_DIR}"; then
    echo -e "${GREEN}✓ Chart linting passed${NC}"
else
    echo -e "${RED}✗ Chart linting failed${NC}"
    exit 1
fi
echo ""

# Template chart (dry-run)
echo -e "${YELLOW}[3/5] Testing chart rendering...${NC}"
if helm template test "${CHART_DIR}" --dry-run > /dev/null; then
    echo -e "${GREEN}✓ Chart rendering successful${NC}"
else
    echo -e "${RED}✗ Chart rendering failed${NC}"
    exit 1
fi
echo ""

# Package chart
echo -e "${YELLOW}[4/5] Packaging chart...${NC}"
if helm package "${CHART_DIR}" --destination "${DIST_DIR}"; then
    PACKAGE_FILE=$(ls -t "${DIST_DIR}"/*.tgz | head -1)
    PACKAGE_NAME=$(basename "${PACKAGE_FILE}")
    echo -e "${GREEN}✓ Chart packaged: ${PACKAGE_NAME}${NC}"
else
    echo -e "${RED}✗ Chart packaging failed${NC}"
    exit 1
fi
echo ""

# Generate chart values documentation
echo -e "${YELLOW}[5/5] Generating values documentation...${NC}"
if command -v helm-docs >/dev/null 2>&1; then
    cd "${CHART_DIR}" && helm-docs
    echo -e "${GREEN}✓ Values documentation generated${NC}"
else
    echo -e "${YELLOW}! helm-docs not found, skipping documentation generation${NC}"
fi
echo ""

# Display package info
echo "========================================="
echo -e "${GREEN}Chart Packaging Complete!${NC}"
echo "========================================="
echo ""
echo "Package: ${PACKAGE_FILE}"
echo "Size: $(du -h "${PACKAGE_FILE}" | cut -f1)"
echo ""
echo "To install:"
echo "  helm install infra-operator ${PACKAGE_FILE} -n infra-operator --create-namespace"
echo ""
echo "To publish to chart repository:"
echo "  helm repo index ${DIST_DIR} --url https://your-org.github.io/charts"
echo "  # Then commit and push to gh-pages branch"
echo ""
