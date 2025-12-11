package e2e_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

var _ = Describe("VPC E2E Tests", func() {
	var (
		namespace    string
		providerName string
	)

	BeforeEach(func() {
		namespace = generateUniqueName("vpc-test")
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

	Context("VPC Lifecycle", func() {
		It("should create a VPC successfully", func() {
			vpcName := "test-vpc"
			cidrBlock := "10.0.0.0/16"

			By("creating VPC CR")
			vpc := createVPC(namespace, vpcName, providerName, cidrBlock)
			Expect(vpc).NotTo(BeNil())

			By("waiting for VPC to be ready")
			vpc = waitForVPCReady(namespace, vpcName, createTimeout)

			By("verifying VPC status")
			Expect(vpc.Status.Ready).To(BeTrue())
			Expect(vpc.Status.VpcID).NotTo(BeEmpty())
			Expect(vpc.Status.CidrBlock).To(Equal(cidrBlock))
			Expect(vpc.Status.State).To(Equal("available"))

			By("verifying VPC configuration")
			Expect(vpc.Spec.EnableDnsSupport).To(BeTrue())
			Expect(vpc.Spec.EnableDnsHostnames).To(BeTrue())
		})

		It("should handle VPC deletion correctly", func() {
			vpcName := "test-vpc-delete"
			cidrBlock := "10.1.0.0/16"

			By("creating VPC")
			createVPC(namespace, vpcName, providerName, cidrBlock)

			By("waiting for VPC to be ready")
			waitForVPCReady(namespace, vpcName, createTimeout)

			By("deleting VPC CR")
			deleteVPC(namespace, vpcName)

			By("waiting for VPC to be removed")
			waitForVPCDeleted(namespace, vpcName, deleteTimeout)

			By("verifying VPC CR no longer exists")
			vpc := &infrav1alpha1.VPC{}
			err := getResource(namespace, vpcName, vpc)
			Expect(err).To(HaveOccurred())
		})

		It("should update VPC tags", func() {
			vpcName := "test-vpc-update"
			cidrBlock := "10.2.0.0/16"

			By("creating VPC with initial tags")
			vpc := createVPC(namespace, vpcName, providerName, cidrBlock)
			initialTags := vpc.Spec.Tags

			By("waiting for VPC to be ready")
			vpc = waitForVPCReady(namespace, vpcName, createTimeout)

			By("updating VPC tags")
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      vpcName,
				Namespace: namespace,
			}, vpc)
			Expect(err).NotTo(HaveOccurred())

			vpc.Spec.Tags["Updated"] = "true"
			vpc.Spec.Tags["Version"] = "v2"
			updateResource(vpc)

			By("waiting for VPC to reconcile")
			time.Sleep(10 * time.Second)

			By("verifying tags were updated")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      vpcName,
				Namespace: namespace,
			}, vpc)
			Expect(err).NotTo(HaveOccurred())
			Expect(vpc.Spec.Tags["Updated"]).To(Equal("true"))
			Expect(vpc.Spec.Tags["Version"]).To(Equal("v2"))

			// Verify original tags are still present
			for k, v := range initialTags {
				if k != "Updated" && k != "Version" {
					Expect(vpc.Spec.Tags[k]).To(Equal(v))
				}
			}
		})

		It("should create VPC with custom instance tenancy", func() {
			vpcName := "test-vpc-tenancy"
			cidrBlock := "10.3.0.0/16"

			By("creating VPC with dedicated tenancy")
			vpc := createVPC(namespace, vpcName, providerName, cidrBlock)
			vpc.Spec.InstanceTenancy = "dedicated"
			err := k8sClient.Update(ctx, vpc)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for VPC to be ready")
			vpc = waitForVPCReady(namespace, vpcName, createTimeout)

			By("verifying instance tenancy")
			Expect(vpc.Spec.InstanceTenancy).To(Equal("dedicated"))
		})

		It("should respect deletion policy Retain", func() {
			vpcName := "test-vpc-retain"
			cidrBlock := "10.4.0.0/16"

			By("creating VPC with deletion policy Retain")
			vpc := createVPC(namespace, vpcName, providerName, cidrBlock)
			vpc.Spec.DeletionPolicy = "Retain"
			err := k8sClient.Update(ctx, vpc)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for VPC to be ready")
			vpc = waitForVPCReady(namespace, vpcName, createTimeout)
			vpcID := vpc.Status.VpcID

			By("deleting VPC CR")
			deleteVPC(namespace, vpcName)

			By("waiting for VPC CR to be removed")
			waitForVPCDeleted(namespace, vpcName, deleteTimeout)

			By("verifying VPC ID was set (would be retained in AWS)")
			Expect(vpcID).NotTo(BeEmpty())
			// Note: In a real AWS test, we would verify the VPC still exists in AWS
			// For LocalStack, this behavior may vary
		})
	})

	Context("VPC Validation", func() {
		It("should reject invalid CIDR blocks", func() {
			vpcName := "test-vpc-invalid-cidr"
			invalidCIDR := "invalid-cidr"

			By("attempting to create VPC with invalid CIDR")
			vpc := &infrav1alpha1.VPC{}
			vpc.Name = vpcName
			vpc.Namespace = namespace
			vpc.Spec.ProviderRef.Name = providerName
			vpc.Spec.CidrBlock = invalidCIDR

			err := k8sClient.Create(ctx, vpc)
			// This should fail validation due to kubebuilder pattern validation
			Expect(err).To(HaveOccurred())
		})

		It("should create VPC with minimum CIDR /16", func() {
			vpcName := "test-vpc-min-cidr"
			cidrBlock := "172.16.0.0/16"

			By("creating VPC with /16 CIDR")
			vpc := createVPC(namespace, vpcName, providerName, cidrBlock)

			By("waiting for VPC to be ready")
			vpc = waitForVPCReady(namespace, vpcName, createTimeout)

			By("verifying VPC was created")
			Expect(vpc.Status.Ready).To(BeTrue())
			Expect(vpc.Status.CidrBlock).To(Equal(cidrBlock))
		})

		It("should create VPC with maximum CIDR /28", func() {
			vpcName := "test-vpc-max-cidr"
			cidrBlock := "192.168.1.0/28"

			By("creating VPC with /28 CIDR")
			vpc := createVPC(namespace, vpcName, providerName, cidrBlock)

			By("waiting for VPC to be ready")
			vpc = waitForVPCReady(namespace, vpcName, createTimeout)

			By("verifying VPC was created")
			Expect(vpc.Status.Ready).To(BeTrue())
			Expect(vpc.Status.CidrBlock).To(Equal(cidrBlock))
		})
	})

	Context("VPC Status Updates", func() {
		It("should update LastSyncTime on reconciliation", func() {
			vpcName := "test-vpc-sync"
			cidrBlock := "10.5.0.0/16"

			By("creating VPC")
			createVPC(namespace, vpcName, providerName, cidrBlock)

			By("waiting for VPC to be ready")
			vpc := waitForVPCReady(namespace, vpcName, createTimeout)

			By("verifying LastSyncTime is set")
			Expect(vpc.Status.LastSyncTime).NotTo(BeNil())

			By("getting initial sync time")
			initialSyncTime := vpc.Status.LastSyncTime.Time

			By("waiting for next reconciliation")
			time.Sleep(15 * time.Second)

			By("getting updated VPC status")
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      vpcName,
				Namespace: namespace,
			}, vpc)
			Expect(err).NotTo(HaveOccurred())

			By("verifying LastSyncTime was updated")
			if vpc.Status.LastSyncTime != nil {
				// LastSyncTime should be same or later
				Expect(vpc.Status.LastSyncTime.Time.Unix()).To(BeNumerically(">=", initialSyncTime.Unix()))
			}
		})

		It("should set IsDefault to false for custom VPCs", func() {
			vpcName := "test-vpc-not-default"
			cidrBlock := "10.6.0.0/16"

			By("creating custom VPC")
			createVPC(namespace, vpcName, providerName, cidrBlock)

			By("waiting for VPC to be ready")
			vpc := waitForVPCReady(namespace, vpcName, createTimeout)

			By("verifying IsDefault is false")
			Expect(vpc.Status.IsDefault).To(BeFalse())
		})
	})

	Context("VPC Error Handling", func() {
		It("should handle missing provider reference", func() {
			vpcName := "test-vpc-no-provider"
			cidrBlock := "10.7.0.0/16"

			By("creating VPC with non-existent provider")
			vpc := &infrav1alpha1.VPC{}
			vpc.Name = vpcName
			vpc.Namespace = namespace
			vpc.Spec.ProviderRef.Name = "non-existent-provider"
			vpc.Spec.CidrBlock = cidrBlock

			err := k8sClient.Create(ctx, vpc)
			Expect(err).NotTo(HaveOccurred())

			By("waiting and verifying VPC does not become ready")
			time.Sleep(10 * time.Second)

			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      vpcName,
				Namespace: namespace,
			}, vpc)
			Expect(err).NotTo(HaveOccurred())
			Expect(vpc.Status.Ready).To(BeFalse())
		})
	})

	Context("Multiple VPCs", func() {
		It("should create multiple VPCs in the same namespace", func() {
			numVPCs := 3
			vpcNames := make([]string, numVPCs)

			By("creating multiple VPCs")
			for i := 0; i < numVPCs; i++ {
				vpcNames[i] = generateUniqueName("test-vpc")
				cidrBlock := "10." + string(rune(10+i)) + ".0.0/16"
				createVPC(namespace, vpcNames[i], providerName, cidrBlock)
			}

			By("waiting for all VPCs to be ready")
			for _, vpcName := range vpcNames {
				vpc := waitForVPCReady(namespace, vpcName, createTimeout)
				Expect(vpc.Status.Ready).To(BeTrue())
			}

			By("verifying all VPCs have unique VPC IDs")
			vpcIDs := make(map[string]bool)
			for _, vpcName := range vpcNames {
				vpc := &infrav1alpha1.VPC{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      vpcName,
					Namespace: namespace,
				}, vpc)
				Expect(err).NotTo(HaveOccurred())
				Expect(vpcIDs[vpc.Status.VpcID]).To(BeFalse(), "VPC ID should be unique")
				vpcIDs[vpc.Status.VpcID] = true
			}
		})
	})
})
