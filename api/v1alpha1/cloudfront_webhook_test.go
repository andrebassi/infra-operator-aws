// Package v1alpha1 contém as definições de API para aws-infra-operator.runner.codes/v1alpha1.
//
// Este package define todos os Custom Resource Definitions (CRDs) para gerenciamento
// de recursos AWS através do Kubernetes.
package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Ucloudfront Webhook", func() {
	var obj *CloudFront

	BeforeEach(func() {
		obj = &CloudFront{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cloudfront",
				Namespace: "default",
			},
			Spec: CloudFrontSpec{
				ProviderRef: ProviderReference{Name: "test-provider"},
			},
		}
	})

	Context("ValidateCreate", func() {
		It("should accept valid Ucloudfront", func() {
			warnings, err := obj.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).NotTo(BeEmpty())
		})

		It("should reject empty ProviderRef", func() {
			obj.Spec.ProviderRef.Name = ""
			_, err := obj.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should reject aws: prefix in tags", func() {
			obj.Spec.Tags = map[string]string{"aws:test": "value"}
			_, err := obj.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})
	})
})
