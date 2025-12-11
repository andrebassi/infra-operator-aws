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

var _ = Describe("VPC Webhook", func() {
	var vpc *VPC

	BeforeEach(func() {
		vpc = &VPC{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vpc",
				Namespace: "default",
			},
			Spec: VPCSpec{
				ProviderRef: ProviderReference{Name: "test-provider"},
				CidrBlock:   "10.0.0.0/16",
			},
		}
	})

	Context("ValidateCreate", func() {
		It("should accept valid VPC", func() {
			warnings, err := vpc.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).NotTo(BeEmpty()) // Warning sobre deletionPolicy
		})

		It("should reject invalid CIDR - too small mask", func() {
			vpc.Spec.CidrBlock = "10.0.0.0/8"
			_, err := vpc.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be between /16 and /28"))
		})

		It("should reject invalid CIDR - too large mask", func() {
			vpc.Spec.CidrBlock = "10.0.0.0/29"
			_, err := vpc.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be between /16 and /28"))
		})

		It("should reject malformed CIDR", func() {
			vpc.Spec.CidrBlock = "invalid"
			_, err := vpc.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid CIDR block"))
		})

		It("should reject empty ProviderRef", func() {
			vpc.Spec.ProviderRef.Name = ""
			_, err := vpc.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should reject aws: prefix in tags", func() {
			vpc.Spec.Tags = map[string]string{"aws:test": "value"}
			_, err := vpc.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("ValidateUpdate", func() {
		It("should reject CIDR block change", func() {
			oldVPC := vpc.DeepCopy()
			vpc.Spec.CidrBlock = "10.1.0.0/16"

			_, err := vpc.ValidateUpdate(oldVPC)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})

		It("should allow tag changes", func() {
			oldVPC := vpc.DeepCopy()
			vpc.Spec.Tags = map[string]string{"environment": "production"}

			_, err := vpc.ValidateUpdate(oldVPC)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
