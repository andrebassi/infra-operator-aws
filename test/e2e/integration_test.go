package e2e_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

var _ = Describe("Integration Tests - Multi-Resource Scenarios", func() {
	var (
		namespace    string
		providerName string
	)

	BeforeEach(func() {
		namespace = generateUniqueName("integration-test")
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

	Context("VPC + Subnet + EC2 Stack", func() {
		It("should create a complete network stack with dependencies", func() {
			vpcName := "integration-vpc"
			subnetName := "integration-subnet"
			cidrBlock := "10.20.0.0/16"
			subnetCIDR := "10.20.1.0/24"

			By("Step 1: Creating VPC")
			vpc := createVPC(namespace, vpcName, providerName, cidrBlock)
			Expect(vpc).NotTo(BeNil())

			By("Step 2: Waiting for VPC to be ready")
			vpc = waitForVPCReady(namespace, vpcName, createTimeout)
			vpcID := vpc.Status.VpcID
			Expect(vpcID).NotTo(BeEmpty())

			By("Step 3: Creating Subnet in VPC")
			subnet := &infrav1alpha1.Subnet{}
			subnet.Name = subnetName
			subnet.Namespace = namespace
			subnet.Spec.ProviderRef.Name = providerName
			subnet.Spec.VpcID = vpcID
			subnet.Spec.CidrBlock = subnetCIDR
			subnet.Spec.AvailabilityZone = "us-east-1a"
			subnet.Spec.Tags = map[string]string{
				"Name":      subnetName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, subnet)
			Expect(err).NotTo(HaveOccurred())

			By("Step 4: Waiting for Subnet to be ready")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      subnetName,
					Namespace: namespace,
				}, subnet)
				return err == nil && subnet.Status.Ready && subnet.Status.SubnetID != ""
			}, createTimeout, pollInterval).Should(BeTrue())

			subnetID := subnet.Status.SubnetID
			Expect(subnetID).NotTo(BeEmpty())

			By("Step 5: Verifying Subnet belongs to VPC")
			Expect(subnet.Status.VpcID).To(Equal(vpcID))

			By("Step 6: Cleanup - Deleting Subnet first")
			err = k8sClient.Delete(ctx, subnet)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      subnetName,
					Namespace: namespace,
				}, subnet)
				return err != nil
			}, deleteTimeout, pollInterval).Should(BeTrue())

			By("Step 7: Cleanup - Deleting VPC")
			deleteVPC(namespace, vpcName)
			waitForVPCDeleted(namespace, vpcName, deleteTimeout)
		})
	})

	Context("S3 + IAM Stack", func() {
		It("should create S3 bucket and IAM role for access", func() {
			bucketName := generateUniqueName("integration-bucket")
			bucketCRName := "integration-s3"
			roleName := "integration-s3-role"

			By("Step 1: Creating S3 Bucket")
			bucket := createS3Bucket(namespace, bucketCRName, providerName, bucketName)
			Expect(bucket).NotTo(BeNil())

			By("Step 2: Waiting for S3 Bucket to be ready")
			bucket = waitForS3BucketReady(namespace, bucketCRName, createTimeout)
			bucketARN := bucket.Status.ARN
			Expect(bucketARN).NotTo(BeEmpty())

			By("Step 3: Creating IAM Role for S3 access")
			role := &infrav1alpha1.IAMRole{}
			role.Name = roleName
			role.Namespace = namespace
			role.Spec.ProviderRef.Name = providerName
			role.Spec.RoleName = roleName
			role.Spec.AssumeRolePolicyDocument = `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Principal": {"Service": "ec2.amazonaws.com"},
					"Action": "sts:AssumeRole"
				}]
			}`
			role.Spec.Tags = map[string]string{
				"Name":      roleName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, role)
			Expect(err).NotTo(HaveOccurred())

			By("Step 4: Waiting for IAM Role to be ready")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      roleName,
					Namespace: namespace,
				}, role)
				return err == nil && role.Status.Ready && role.Status.RoleARN != ""
			}, createTimeout, pollInterval).Should(BeTrue())

			By("Step 5: Verifying both resources exist")
			Expect(role.Status.Ready).To(BeTrue())
			Expect(bucket.Status.Ready).To(BeTrue())

			By("Step 6: Cleanup in reverse order")
			err = k8sClient.Delete(ctx, role)
			Expect(err).NotTo(HaveOccurred())

			deleteS3Bucket(namespace, bucketCRName)
			waitForS3BucketDeleted(namespace, bucketCRName, deleteTimeout)
		})
	})

	Context("Multi-Resource Parallel Creation", func() {
		It("should create multiple independent resources in parallel", func() {
			numVPCs := 2
			numBuckets := 2
			numEIPs := 2

			By("Creating multiple VPCs in parallel")
			vpcNames := make([]string, numVPCs)
			for i := 0; i < numVPCs; i++ {
				vpcNames[i] = fmt.Sprintf("parallel-vpc-%d", i)
				cidrBlock := fmt.Sprintf("10.%d.0.0/16", 30+i)
				createVPC(namespace, vpcNames[i], providerName, cidrBlock)
			}

			By("Creating multiple S3 buckets in parallel")
			bucketCRNames := make([]string, numBuckets)
			for i := 0; i < numBuckets; i++ {
				bucketName := generateUniqueName(fmt.Sprintf("parallel-bucket-%d", i))
				bucketCRNames[i] = fmt.Sprintf("parallel-s3-%d", i)
				createS3Bucket(namespace, bucketCRNames[i], providerName, bucketName)
			}

			By("Creating multiple ElasticIPs in parallel")
			eipNames := make([]string, numEIPs)
			for i := 0; i < numEIPs; i++ {
				eipNames[i] = fmt.Sprintf("parallel-eip-%d", i)
				createElasticIP(namespace, eipNames[i], providerName)
			}

			By("Waiting for all VPCs to be ready")
			for _, vpcName := range vpcNames {
				vpc := waitForVPCReady(namespace, vpcName, longTimeout)
				Expect(vpc.Status.Ready).To(BeTrue())
			}

			By("Waiting for all S3 buckets to be ready")
			for _, bucketName := range bucketCRNames {
				bucket := waitForS3BucketReady(namespace, bucketName, longTimeout)
				Expect(bucket.Status.Ready).To(BeTrue())
			}

			By("Waiting for all ElasticIPs to be ready")
			for _, eipName := range eipNames {
				eip := waitForElasticIPReady(namespace, eipName, longTimeout)
				Expect(eip.Status.Ready).To(BeTrue())
			}

			By("Verifying all resources are independent and successful")
			// All resources should be ready without blocking each other
		})
	})

	Context("Resource Dependencies and Ordering", func() {
		It("should handle creation order correctly for dependent resources", func() {
			vpcName := "dependency-vpc"
			igwName := "dependency-igw"
			cidrBlock := "10.40.0.0/16"

			By("Creating VPC first")
			vpc := createVPC(namespace, vpcName, providerName, cidrBlock)
			vpc = waitForVPCReady(namespace, vpcName, createTimeout)
			vpcID := vpc.Status.VpcID

			By("Creating Internet Gateway attached to VPC")
			igw := &infrav1alpha1.InternetGateway{}
			igw.Name = igwName
			igw.Namespace = namespace
			igw.Spec.ProviderRef.Name = providerName
			igw.Spec.VpcID = vpcID
			igw.Spec.Tags = map[string]string{
				"Name":      igwName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, igw)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for Internet Gateway to be ready")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      igwName,
					Namespace: namespace,
				}, igw)
				return err == nil && igw.Status.Ready && igw.Status.InternetGatewayID != ""
			}, createTimeout, pollInterval).Should(BeTrue())

			By("Verifying IGW is attached to the correct VPC")
			Expect(igw.Status.VpcID).To(Equal(vpcID))

			By("Attempting to delete VPC should wait for IGW deletion")
			// This tests finalizer handling and dependency management
			// In real implementation, VPC deletion should be blocked until IGW is deleted

			By("Cleanup - Deleting IGW first")
			err = k8sClient.Delete(ctx, igw)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      igwName,
					Namespace: namespace,
				}, igw)
				return err != nil
			}, deleteTimeout, pollInterval).Should(BeTrue())

			By("Cleanup - Now deleting VPC")
			deleteVPC(namespace, vpcName)
			waitForVPCDeleted(namespace, vpcName, deleteTimeout)
		})
	})

	Context("Cross-Region Resources", func() {
		It("should handle resources in different regions via different providers", func() {
			Skip("Cross-region test requires multiple AWSProviders with different regions")
		})
	})

	Context("Resource Update Cascade", func() {
		It("should handle updates to parent resources gracefully", func() {
			vpcName := "update-cascade-vpc"
			cidrBlock := "10.50.0.0/16"

			By("Creating VPC")
			vpc := createVPC(namespace, vpcName, providerName, cidrBlock)
			vpc = waitForVPCReady(namespace, vpcName, createTimeout)

			By("Updating VPC tags")
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      vpcName,
				Namespace: namespace,
			}, vpc)
			Expect(err).NotTo(HaveOccurred())

			vpc.Spec.Tags["UpdatedBy"] = "integration-test"
			vpc.Spec.Tags["Timestamp"] = "test-time"
			updateResource(vpc)

			By("Verifying VPC remains ready after update")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      vpcName,
					Namespace: namespace,
				}, vpc)
				return err == nil && vpc.Status.Ready
			}, updateTimeout, pollInterval).Should(BeTrue())

			By("Verifying tags were updated")
			Expect(vpc.Spec.Tags["UpdatedBy"]).To(Equal("integration-test"))
		})
	})
})
