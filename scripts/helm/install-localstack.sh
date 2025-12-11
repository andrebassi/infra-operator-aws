#!/usr/bin/env bash
#
# Quick install for LocalStack testing
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CHART_DIR="${REPO_ROOT}/chart"

NAMESPACE="${1:-infra-operator}"
RELEASE_NAME="${2:-infra-operator}"

echo "Installing infra-operator for LocalStack..."
echo ""
echo "Namespace: ${NAMESPACE}"
echo "Release: ${RELEASE_NAME}"
echo ""

# Ensure LocalStack is running
if ! kubectl get pod localstack -n default > /dev/null 2>&1; then
    echo "WARNING: LocalStack pod not found in default namespace"
    echo "Start LocalStack first:"
    echo "  docker-compose up -d localstack"
    echo ""
fi

# Create namespace
kubectl create namespace "${NAMESPACE}" 2>/dev/null || true

# Install with localstack values
helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  --values "${CHART_DIR}/values-localstack.yaml" \
  --wait \
  --timeout 5m

echo ""
echo "Installation complete!"
echo ""
echo "Check status:"
echo "  kubectl get pods -n ${NAMESPACE}"
echo "  kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/name=infra-operator -f"
echo ""
echo "Test with:"
echo "  kubectl apply -f config/samples/01-awsprovider.yaml"
