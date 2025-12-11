#!/usr/bin/env bash
#
# Quick install for development environment
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CHART_DIR="${REPO_ROOT}/chart"

NAMESPACE="${1:-infra-operator}"
RELEASE_NAME="${2:-infra-operator}"

echo "Installing infra-operator for development..."
echo ""
echo "Namespace: ${NAMESPACE}"
echo "Release: ${RELEASE_NAME}"
echo ""

# Create namespace
kubectl create namespace "${NAMESPACE}" 2>/dev/null || true

# Install with dev values
helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  --values "${CHART_DIR}/values-dev.yaml" \
  --wait \
  --timeout 5m

echo ""
echo "Installation complete!"
echo ""
echo "Check status:"
echo "  kubectl get pods -n ${NAMESPACE}"
echo "  kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/name=infra-operator -f"
