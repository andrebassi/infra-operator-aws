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

var _ = Describe("S3Bucket Webhook", func() {
	var bucket *S3Bucket

	BeforeEach(func() {
		bucket = &S3Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bucket",
				Namespace: "default",
			},
			Spec: S3BucketSpec{
				ProviderRef: ProviderReference{Name: "test-provider"},
				BucketName:  "my-test-bucket-123",
			},
		}
	})

	Context("ValidateCreate", func() {
		It("should accept valid bucket", func() {
			_, err := bucket.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject bucket name too short", func() {
			bucket.Spec.BucketName = "ab"
			_, err := bucket.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("3-63 characters"))
		})

		It("should reject bucket name too long", func() {
			bucket.Spec.BucketName = "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"
			_, err := bucket.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should reject uppercase in bucket name", func() {
			bucket.Spec.BucketName = "MyBucket"
			_, err := bucket.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should reject IP address format", func() {
			bucket.Spec.BucketName = "192.168.1.1"
			_, err := bucket.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("IP address"))
		})

		It("should reject consecutive periods", func() {
			bucket.Spec.BucketName = "my..bucket"
			_, err := bucket.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("consecutive periods"))
		})

		It("should reject xn-- prefix", func() {
			bucket.Spec.BucketName = "xn--mybucket"
			_, err := bucket.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should reject -s3alias suffix", func() {
			bucket.Spec.BucketName = "mybucket-s3alias"
			_, err := bucket.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should require KMS key ID when using aws_kms", func() {
			bucket.Spec.Encryption = &ServerSideEncryptionConfiguration{
				Algorithm: "aws_kms",
			}
			_, err := bucket.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("kmsKeyID is required"))
		})

		It("should reject lifecycle rule without ID", func() {
			bucket.Spec.LifecycleRules = []LifecycleRule{
				{
					Enabled: true,
					Expiration: &LifecycleExpiration{
						Days: 30,
					},
				},
			}
			_, err := bucket.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("id is required"))
		})
	})

	Context("ValidateUpdate", func() {
		It("should reject bucket name change", func() {
			oldBucket := bucket.DeepCopy()
			bucket.Spec.BucketName = "new-bucket-name"

			_, err := bucket.ValidateUpdate(oldBucket)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})
	})
})
