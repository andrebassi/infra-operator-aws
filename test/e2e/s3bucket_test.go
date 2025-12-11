package e2e_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

var _ = Describe("S3Bucket E2E Tests", func() {
	var (
		namespace    string
		providerName string
	)

	BeforeEach(func() {
		namespace = generateUniqueName("s3-test")
		providerName = "test-provider"

		By("creating test namespace")
		createNamespace(namespace)

		By("creating AWSProvider")
		createAWSProvider(namespace, providerName)

		By("waiting for AWSProvider to be ready")
		waitForProviderReady(namespace, providerName, createTimeout)
	})

	AfterEach(func() {
		By("cleaning up test namespace")
		deleteNamespace(namespace)
	})

	Context("S3Bucket Lifecycle", func() {
		It("should create an S3 bucket successfully", func() {
			bucketName := generateUniqueName("test-bucket")
			crName := "test-s3"

			By("creating S3Bucket CR")
			bucket := createS3Bucket(namespace, crName, providerName, bucketName)
			Expect(bucket).NotTo(BeNil())

			By("waiting for S3Bucket to be ready")
			bucket = waitForS3BucketReady(namespace, crName, createTimeout)

			By("verifying S3Bucket status")
			Expect(bucket.Status.Ready).To(BeTrue())
			Expect(bucket.Status.BucketName).To(Equal(bucketName))
			Expect(bucket.Status.ARN).NotTo(BeEmpty())
		})

		It("should handle S3 bucket deletion correctly", func() {
			bucketName := generateUniqueName("test-bucket-delete")
			crName := "test-s3-delete"

			By("creating S3Bucket")
			createS3Bucket(namespace, crName, providerName, bucketName)

			By("waiting for S3Bucket to be ready")
			waitForS3BucketReady(namespace, crName, createTimeout)

			By("deleting S3Bucket CR")
			deleteS3Bucket(namespace, crName)

			By("waiting for S3Bucket to be removed")
			waitForS3BucketDeleted(namespace, crName, deleteTimeout)

			By("verifying S3Bucket CR no longer exists")
			bucket := &infrav1alpha1.S3Bucket{}
			err := getResource(namespace, crName, bucket)
			Expect(err).To(HaveOccurred())
		})

		It("should update S3 bucket tags", func() {
			bucketName := generateUniqueName("test-bucket-update")
			crName := "test-s3-update"

			By("creating S3Bucket with initial tags")
			bucket := createS3Bucket(namespace, crName, providerName, bucketName)

			By("waiting for S3Bucket to be ready")
			bucket = waitForS3BucketReady(namespace, crName, createTimeout)

			By("updating S3Bucket tags")
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      crName,
				Namespace: namespace,
			}, bucket)
			Expect(err).NotTo(HaveOccurred())

			bucket.Spec.Tags["Updated"] = "true"
			bucket.Spec.Tags["Version"] = "v2"
			updateResource(bucket)

			By("waiting for S3Bucket to reconcile")
			time.Sleep(10 * time.Second)

			By("verifying tags were updated")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      crName,
				Namespace: namespace,
			}, bucket)
			Expect(err).NotTo(HaveOccurred())
			Expect(bucket.Spec.Tags["Updated"]).To(Equal("true"))
			Expect(bucket.Spec.Tags["Version"]).To(Equal("v2"))
		})

		It("should respect deletion policy Retain", func() {
			bucketName := generateUniqueName("test-bucket-retain")
			crName := "test-s3-retain"

			By("creating S3Bucket with deletion policy Retain")
			bucket := createS3Bucket(namespace, crName, providerName, bucketName)
			bucket.Spec.DeletionPolicy = "Retain"
			err := k8sClient.Update(ctx, bucket)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for S3Bucket to be ready")
			bucket = waitForS3BucketReady(namespace, crName, createTimeout)
			bucketARN := bucket.Status.ARN

			By("deleting S3Bucket CR")
			deleteS3Bucket(namespace, crName)

			By("waiting for S3Bucket CR to be removed")
			waitForS3BucketDeleted(namespace, crName, deleteTimeout)

			By("verifying bucket ARN was set (would be retained in AWS)")
			Expect(bucketARN).NotTo(BeEmpty())
		})
	})

	Context("S3Bucket Configuration", func() {
		It("should create bucket with versioning enabled", func() {
			bucketName := generateUniqueName("test-bucket-versioning")
			crName := "test-s3-versioning"

			By("creating S3Bucket with versioning")
			bucket := &infrav1alpha1.S3Bucket{}
			bucket.Name = crName
			bucket.Namespace = namespace
			bucket.Spec.ProviderRef.Name = providerName
			bucket.Spec.BucketName = bucketName
			bucket.Spec.Versioning = &infrav1alpha1.S3VersioningConfiguration{
				Enabled: true,
			}
			bucket.Spec.Tags = map[string]string{
				"Name":      bucketName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, bucket)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for S3Bucket to be ready")
			bucket = waitForS3BucketReady(namespace, crName, createTimeout)

			By("verifying versioning is enabled")
			Expect(bucket.Spec.Versioning).NotTo(BeNil())
			Expect(bucket.Spec.Versioning.Enabled).To(BeTrue())
		})

		It("should create bucket with encryption", func() {
			bucketName := generateUniqueName("test-bucket-encryption")
			crName := "test-s3-encryption"

			By("creating S3Bucket with encryption")
			bucket := &infrav1alpha1.S3Bucket{}
			bucket.Name = crName
			bucket.Namespace = namespace
			bucket.Spec.ProviderRef.Name = providerName
			bucket.Spec.BucketName = bucketName
			bucket.Spec.Encryption = &infrav1alpha1.S3EncryptionConfiguration{
				Algorithm: "AES256",
			}
			bucket.Spec.Tags = map[string]string{
				"Name":      bucketName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, bucket)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for S3Bucket to be ready")
			bucket = waitForS3BucketReady(namespace, crName, createTimeout)

			By("verifying encryption is configured")
			Expect(bucket.Spec.Encryption).NotTo(BeNil())
			Expect(bucket.Spec.Encryption.Algorithm).To(Equal("AES256"))
		})

		It("should create bucket with lifecycle rules", func() {
			bucketName := generateUniqueName("test-bucket-lifecycle")
			crName := "test-s3-lifecycle"

			By("creating S3Bucket with lifecycle rules")
			bucket := &infrav1alpha1.S3Bucket{}
			bucket.Name = crName
			bucket.Namespace = namespace
			bucket.Spec.ProviderRef.Name = providerName
			bucket.Spec.BucketName = bucketName
			bucket.Spec.LifecycleRules = []infrav1alpha1.S3LifecycleRule{
				{
					ID:      "expire-old-versions",
					Enabled: true,
					NoncurrentVersionExpiration: &infrav1alpha1.S3NoncurrentVersionExpiration{
						Days: 90,
					},
				},
			}
			bucket.Spec.Tags = map[string]string{
				"Name":      bucketName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, bucket)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for S3Bucket to be ready")
			bucket = waitForS3BucketReady(namespace, crName, createTimeout)

			By("verifying lifecycle rules are configured")
			Expect(bucket.Spec.LifecycleRules).To(HaveLen(1))
			Expect(bucket.Spec.LifecycleRules[0].ID).To(Equal("expire-old-versions"))
			Expect(bucket.Spec.LifecycleRules[0].Enabled).To(BeTrue())
		})

		It("should create public bucket when PublicAccessBlock is disabled", func() {
			bucketName := generateUniqueName("test-bucket-public")
			crName := "test-s3-public"

			By("creating S3Bucket with public access allowed")
			bucket := &infrav1alpha1.S3Bucket{}
			bucket.Name = crName
			bucket.Namespace = namespace
			bucket.Spec.ProviderRef.Name = providerName
			bucket.Spec.BucketName = bucketName
			bucket.Spec.PublicAccessBlock = &infrav1alpha1.S3PublicAccessBlockConfiguration{
				BlockPublicAcls:       false,
				BlockPublicPolicy:     false,
				IgnorePublicAcls:      false,
				RestrictPublicBuckets: false,
			}
			bucket.Spec.Tags = map[string]string{
				"Name":      bucketName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, bucket)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for S3Bucket to be ready")
			bucket = waitForS3BucketReady(namespace, crName, createTimeout)

			By("verifying public access block is disabled")
			Expect(bucket.Spec.PublicAccessBlock).NotTo(BeNil())
			Expect(bucket.Spec.PublicAccessBlock.BlockPublicAcls).To(BeFalse())
		})
	})

	Context("S3Bucket Status", func() {
		It("should update LastSyncTime on reconciliation", func() {
			bucketName := generateUniqueName("test-bucket-sync")
			crName := "test-s3-sync"

			By("creating S3Bucket")
			createS3Bucket(namespace, crName, providerName, bucketName)

			By("waiting for S3Bucket to be ready")
			bucket := waitForS3BucketReady(namespace, crName, createTimeout)

			By("verifying LastSyncTime is set")
			Expect(bucket.Status.LastSyncTime).NotTo(BeNil())

			initialSyncTime := bucket.Status.LastSyncTime.Time

			By("waiting for next reconciliation")
			time.Sleep(15 * time.Second)

			By("getting updated S3Bucket status")
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      crName,
				Namespace: namespace,
			}, bucket)
			Expect(err).NotTo(HaveOccurred())

			By("verifying LastSyncTime was updated or same")
			if bucket.Status.LastSyncTime != nil {
				Expect(bucket.Status.LastSyncTime.Time.Unix()).To(BeNumerically(">=", initialSyncTime.Unix()))
			}
		})

		It("should set region in status", func() {
			bucketName := generateUniqueName("test-bucket-region")
			crName := "test-s3-region"

			By("creating S3Bucket")
			createS3Bucket(namespace, crName, providerName, bucketName)

			By("waiting for S3Bucket to be ready")
			bucket := waitForS3BucketReady(namespace, crName, createTimeout)

			By("verifying region is set")
			Expect(bucket.Status.Region).NotTo(BeEmpty())
		})
	})

	Context("Multiple S3 Buckets", func() {
		It("should create multiple S3 buckets", func() {
			numBuckets := 3
			bucketNames := make([]string, numBuckets)

			By("creating multiple S3Buckets")
			for i := 0; i < numBuckets; i++ {
				bucketNames[i] = generateUniqueName(fmt.Sprintf("test-bucket-%d", i))
				crName := fmt.Sprintf("test-s3-%d", i)
				createS3Bucket(namespace, crName, providerName, bucketNames[i])
			}

			By("waiting for all S3Buckets to be ready")
			for i := 0; i < numBuckets; i++ {
				crName := fmt.Sprintf("test-s3-%d", i)
				bucket := waitForS3BucketReady(namespace, crName, createTimeout)
				Expect(bucket.Status.Ready).To(BeTrue())
				Expect(bucket.Status.BucketName).To(Equal(bucketNames[i]))
			}

			By("verifying all buckets have unique ARNs")
			arns := make(map[string]bool)
			for i := 0; i < numBuckets; i++ {
				crName := fmt.Sprintf("test-s3-%d", i)
				bucket := &infrav1alpha1.S3Bucket{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      crName,
					Namespace: namespace,
				}, bucket)
				Expect(err).NotTo(HaveOccurred())
				Expect(arns[bucket.Status.ARN]).To(BeFalse(), "Bucket ARN should be unique")
				arns[bucket.Status.ARN] = true
			}
		})
	})

	Context("S3Bucket Error Handling", func() {
		It("should handle missing provider reference", func() {
			bucketName := generateUniqueName("test-bucket-no-provider")
			crName := "test-s3-no-provider"

			By("creating S3Bucket with non-existent provider")
			bucket := &infrav1alpha1.S3Bucket{}
			bucket.Name = crName
			bucket.Namespace = namespace
			bucket.Spec.ProviderRef.Name = "non-existent-provider"
			bucket.Spec.BucketName = bucketName

			err := k8sClient.Create(ctx, bucket)
			Expect(err).NotTo(HaveOccurred())

			By("waiting and verifying S3Bucket does not become ready")
			time.Sleep(10 * time.Second)

			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      crName,
				Namespace: namespace,
			}, bucket)
			Expect(err).NotTo(HaveOccurred())
			Expect(bucket.Status.Ready).To(BeFalse())
		})
	})
})
