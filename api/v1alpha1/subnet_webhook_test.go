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

var _ = Describe("Subnet Webhook", func() {
	var subnet *Subnet

	BeforeEach(func() {
		subnet = &Subnet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-subnet",
				Namespace: "default",
			},
			Spec: SubnetSpec{
				ProviderRef:      ProviderReference{Name: "test-provider"},
				VpcID:            "vpc-1234567890abcdef0",
				CidrBlock:        "10.0.1.0/24",
				AvailabilityZone: "us-east-1a",
			},
		}
	})

	Context("ValidateCreate", func() {
		It("should accept valid subnet", func() {
			warnings, err := subnet.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).NotTo(BeEmpty()) // Warning sobre deletionPolicy
		})

		It("should reject invalid CIDR - too small mask", func() {
			subnet.Spec.CidrBlock = "10.0.0.0/8"
			_, err := subnet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be between /16 and /28"))
		})

		It("should reject invalid CIDR - too large mask", func() {
			subnet.Spec.CidrBlock = "10.0.0.0/29"
			_, err := subnet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be between /16 and /28"))
		})

		It("should reject malformed CIDR", func() {
			subnet.Spec.CidrBlock = "invalid"
			_, err := subnet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid CIDR block"))
		})

		It("should reject empty ProviderRef", func() {
			subnet.Spec.ProviderRef.Name = ""
			_, err := subnet.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should reject invalid VPC ID format", func() {
			subnet.Spec.VpcID = "invalid-vpc-id"
			_, err := subnet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("vpc-"))
		})

		It("should accept valid VPC ID formats", func() {
			validVPCIDs := []string{
				"vpc-12345678",
				"vpc-1234567890abcdef0",
				"vpc-abc123def456",
			}

			for _, vpcID := range validVPCIDs {
				subnet.Spec.VpcID = vpcID
				_, err := subnet.ValidateCreate()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should reject aws: prefix in tags", func() {
			subnet.Spec.Tags = map[string]string{"aws:test": "value"}
			_, err := subnet.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should accept valid tags", func() {
			subnet.Spec.Tags = map[string]string{
				"Name":        "public-subnet",
				"Environment": "production",
				"Tier":        "public",
			}
			_, err := subnet.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("ValidateUpdate", func() {
		It("should reject CIDR block change", func() {
			oldSubnet := subnet.DeepCopy()
			subnet.Spec.CidrBlock = "10.0.2.0/24"

			_, err := subnet.ValidateUpdate(oldSubnet)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})

		It("should reject VPC ID change", func() {
			oldSubnet := subnet.DeepCopy()
			subnet.Spec.VpcID = "vpc-0987654321fedcba0"

			_, err := subnet.ValidateUpdate(oldSubnet)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})

		It("should reject availability zone change", func() {
			oldSubnet := subnet.DeepCopy()
			subnet.Spec.AvailabilityZone = "us-east-1b"

			_, err := subnet.ValidateUpdate(oldSubnet)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})

		It("should allow tag changes", func() {
			oldSubnet := subnet.DeepCopy()
			subnet.Spec.Tags = map[string]string{"environment": "production"}

			_, err := subnet.ValidateUpdate(oldSubnet)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow mapPublicIpOnLaunch changes", func() {
			oldSubnet := subnet.DeepCopy()
			subnet.Spec.MapPublicIpOnLaunch = true

			_, err := subnet.ValidateUpdate(oldSubnet)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Edge Cases", func() {
		It("should accept /16 CIDR (minimum)", func() {
			subnet.Spec.CidrBlock = "10.0.0.0/16"
			_, err := subnet.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should accept /28 CIDR (maximum)", func() {
			subnet.Spec.CidrBlock = "10.0.1.0/28"
			_, err := subnet.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should accept subnet without availability zone", func() {
			subnet.Spec.AvailabilityZone = ""
			_, err := subnet.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should accept subnet without tags", func() {
			subnet.Spec.Tags = nil
			_, err := subnet.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
