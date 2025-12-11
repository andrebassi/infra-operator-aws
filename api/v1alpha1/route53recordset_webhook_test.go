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

var _ = Describe("Route53RecordSet Webhook", func() {
	var recordSet *Route53RecordSet

	BeforeEach(func() {
		ttl := int64(300)
		recordSet = &Route53RecordSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-record",
				Namespace: "default",
			},
			Spec: Route53RecordSetSpec{
				ProviderRef:  ProviderReference{Name: "test-provider"},
				HostedZoneID: "Z1234567890ABC",
				Name:         "example.com",
				Type:         "A",
				TTL:          &ttl,
				ResourceRecords: []string{
					"192.0.2.1",
				},
			},
		}
	})

	Context("ValidateCreate", func() {
		It("should accept valid record set", func() {
			_, err := recordSet.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject empty ProviderRef", func() {
			recordSet.Spec.ProviderRef.Name = ""
			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should reject invalid hosted zone ID", func() {
			recordSet.Spec.HostedZoneID = "invalid"
			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("hostedZoneID must start with 'Z'"))
		})

		It("should reject both alias and resource records", func() {
			recordSet.Spec.AliasTarget = &AliasTarget{
				HostedZoneID: "Z123456",
				DNSName:      "example.com",
			}
			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot specify both aliasTarget and resourceRecords"))
		})

		It("should reject alias with TTL", func() {
			recordSet.Spec.ResourceRecords = nil
			recordSet.Spec.AliasTarget = &AliasTarget{
				HostedZoneID: "Z123456",
				DNSName:      "example.com",
			}
			// TTL is still set from BeforeEach
			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot specify TTL with aliasTarget"))
		})

		It("should require TTL for non-alias records", func() {
			recordSet.Spec.TTL = nil
			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ttl is required for non-alias records"))
		})

		It("should require resource records for non-alias", func() {
			recordSet.Spec.ResourceRecords = nil
			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resourceRecords is required"))
		})

		It("should warn about low TTL", func() {
			lowTTL := int64(30)
			recordSet.Spec.TTL = &lowTTL
			warnings, err := recordSet.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).NotTo(BeEmpty())
			Expect(warnings[0]).To(ContainSubstring("TTL"))
		})

		It("should validate MX record format", func() {
			recordSet.Spec.Type = "MX"
			recordSet.Spec.ResourceRecords = []string{"10 mail.example.com"}
			_, err := recordSet.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject invalid MX record format", func() {
			recordSet.Spec.Type = "MX"
			recordSet.Spec.ResourceRecords = []string{"mail.example.com"} // Missing priority
			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should validate TXT record length", func() {
			recordSet.Spec.Type = "TXT"
			longValue := make([]byte, 300)
			for i := range longValue {
				longValue[i] = 'a'
			}
			recordSet.Spec.ResourceRecords = []string{string(longValue)}
			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot exceed 255 characters"))
		})

		It("should reject multiple routing policies", func() {
			weight := int64(100)
			recordSet.Spec.Weight = &weight
			recordSet.Spec.Region = "us-east-1"
			recordSet.Spec.SetIdentifier = "test-id"

			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("only one routing policy"))
		})

		It("should require setIdentifier with routing policy", func() {
			weight := int64(100)
			recordSet.Spec.Weight = &weight

			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("setIdentifier is required"))
		})

		It("should validate weight range", func() {
			weight := int64(300)
			recordSet.Spec.Weight = &weight
			recordSet.Spec.SetIdentifier = "test-id"

			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("weight must be between 0 and 255"))
		})

		It("should validate geolocation", func() {
			recordSet.Spec.GeoLocation = &GeoLocation{}
			recordSet.Spec.SetIdentifier = "test-id"

			_, err := recordSet.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("geolocation must specify"))
		})
	})

	Context("ValidateUpdate", func() {
		It("should reject hosted zone ID change", func() {
			oldRecord := recordSet.DeepCopy()
			recordSet.Spec.HostedZoneID = "Z9876543210ABC"

			_, err := recordSet.ValidateUpdate(oldRecord)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})

		It("should reject name change", func() {
			oldRecord := recordSet.DeepCopy()
			recordSet.Spec.Name = "new.example.com"

			_, err := recordSet.ValidateUpdate(oldRecord)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})

		It("should reject type change", func() {
			oldRecord := recordSet.DeepCopy()
			recordSet.Spec.Type = "AAAA"

			_, err := recordSet.ValidateUpdate(oldRecord)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})

		It("should allow TTL change", func() {
			oldRecord := recordSet.DeepCopy()
			newTTL := int64(600)
			recordSet.Spec.TTL = &newTTL

			_, err := recordSet.ValidateUpdate(oldRecord)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow resource records change", func() {
			oldRecord := recordSet.DeepCopy()
			recordSet.Spec.ResourceRecords = []string{"192.0.2.2"}

			_, err := recordSet.ValidateUpdate(oldRecord)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
