// Package v1alpha1 contém as definições de API para aws-infra-operator.runner.codes/v1alpha1.
//
// Este package define todos os Custom Resource Definitions (CRDs) para gerenciamento
// de recursos AWS através do Kubernetes.
package v1alpha1

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWebhooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
}
