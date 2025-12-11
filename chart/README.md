# Infra Operator Helm Chart

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/infra-operator)](https://artifacthub.io/packages/helm/infra-operator/infra-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.20%2B-blue.svg)](https://kubernetes.io/)

Production-ready Helm chart for deploying the Infra Operator - a Kubernetes operator for managing AWS infrastructure resources.

## Quick Installation

```bash
helm repo add infra-operator https://your-org.github.io/infra-operator
helm install infra-operator infra-operator/infra-operator -n infra-operator --create-namespace
```

For detailed configuration and usage, see the full README at `/Users/andrebassi/works/.solutions/operators/infra-operator/chart/CHART_README.md`

