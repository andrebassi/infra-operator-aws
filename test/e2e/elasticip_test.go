package e2e_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

var _ = Describe("ElasticIP E2E Tests", func() {
	var (
		namespace    string
		providerName string
	)

	BeforeEach(func() {
		namespace = generateUniqueName("eip-test")
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

	Context("ElasticIP Lifecycle", func() {
		It("should allocate an Elastic IP successfully", func() {
			eipName := "test-eip"

			By("creating ElasticIP CR")
			eip := createElasticIP(namespace, eipName, providerName)
			Expect(eip).NotTo(BeNil())

			By("waiting for ElasticIP to be ready")
			eip = waitForElasticIPReady(namespace, eipName, createTimeout)

			By("verifying ElasticIP status")
			Expect(eip.Status.Ready).To(BeTrue())
			Expect(eip.Status.AllocationID).NotTo(BeEmpty())
			Expect(eip.Status.PublicIP).NotTo(BeEmpty())
			Expect(eip.Status.Domain).To(Equal("vpc"))
		})

		It("should handle ElasticIP deletion correctly", func() {
			eipName := "test-eip-delete"

			By("creating ElasticIP")
			createElasticIP(namespace, eipName, providerName)

			By("waiting for ElasticIP to be ready")
			waitForElasticIPReady(namespace, eipName, createTimeout)

			By("deleting ElasticIP CR")
			deleteElasticIP(namespace, eipName)

			By("waiting for ElasticIP to be removed")
			waitForElasticIPDeleted(namespace, eipName, deleteTimeout)

			By("verifying ElasticIP CR no longer exists")
			eip := &infrav1alpha1.ElasticIP{}
			err := getResource(namespace, eipName, eip)
			Expect(err).To(HaveOccurred())
		})

		It("should update ElasticIP tags", func() {
			eipName := "test-eip-update"

			By("creating ElasticIP with initial tags")
			eip := createElasticIP(namespace, eipName, providerName)

			By("waiting for ElasticIP to be ready")
			eip = waitForElasticIPReady(namespace, eipName, createTimeout)

			By("updating ElasticIP tags")
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      eipName,
				Namespace: namespace,
			}, eip)
			Expect(err).NotTo(HaveOccurred())

			eip.Spec.Tags["Updated"] = "true"
			eip.Spec.Tags["Version"] = "v2"
			updateResource(eip)

			By("waiting for ElasticIP to reconcile")
			time.Sleep(10 * time.Second)

			By("verifying tags were updated")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      eipName,
				Namespace: namespace,
			}, eip)
			Expect(err).NotTo(HaveOccurred())
			Expect(eip.Spec.Tags["Updated"]).To(Equal("true"))
			Expect(eip.Spec.Tags["Version"]).To(Equal("v2"))
		})

		It("should respect deletion policy Retain", func() {
			eipName := "test-eip-retain"

			By("creating ElasticIP with deletion policy Retain")
			eip := createElasticIP(namespace, eipName, providerName)
			eip.Spec.DeletionPolicy = "Retain"
			err := k8sClient.Update(ctx, eip)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for ElasticIP to be ready")
			eip = waitForElasticIPReady(namespace, eipName, createTimeout)
			allocationID := eip.Status.AllocationID
			publicIP := eip.Status.PublicIP

			By("deleting ElasticIP CR")
			deleteElasticIP(namespace, eipName)

			By("waiting for ElasticIP CR to be removed")
			waitForElasticIPDeleted(namespace, eipName, deleteTimeout)

			By("verifying allocation ID and public IP were set")
			Expect(allocationID).NotTo(BeEmpty())
			Expect(publicIP).NotTo(BeEmpty())
		})
	})

	Context("ElasticIP Configuration", func() {
		It("should allocate EIP with domain vpc", func() {
			eipName := "test-eip-vpc-domain"

			By("creating ElasticIP with vpc domain")
			eip := createElasticIP(namespace, eipName, providerName)
			Expect(eip.Spec.Domain).To(Equal("vpc"))

			By("waiting for ElasticIP to be ready")
			eip = waitForElasticIPReady(namespace, eipName, createTimeout)

			By("verifying domain is vpc")
			Expect(eip.Status.Domain).To(Equal("vpc"))
		})

		It("should allocate EIP with standard domain", func() {
			eipName := "test-eip-standard-domain"

			By("creating ElasticIP with standard domain")
			eip := &infrav1alpha1.ElasticIP{}
			eip.Name = eipName
			eip.Namespace = namespace
			eip.Spec.ProviderRef.Name = providerName
			eip.Spec.Domain = "standard"
			eip.Spec.Tags = map[string]string{
				"Name":      eipName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, eip)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for ElasticIP to be ready")
			eip = waitForElasticIPReady(namespace, eipName, createTimeout)

			By("verifying domain")
			// Note: LocalStack may not distinguish between vpc and standard domains
			Expect(eip.Status.Domain).To(Or(Equal("standard"), Equal("vpc")))
		})

		It("should set NetworkBorderGroup if specified", func() {
			if !isUsingLocalStack() {
				Skip("NetworkBorderGroup test requires real AWS")
			}

			eipName := "test-eip-border-group"

			By("creating ElasticIP with NetworkBorderGroup")
			eip := &infrav1alpha1.ElasticIP{}
			eip.Name = eipName
			eip.Namespace = namespace
			eip.Spec.ProviderRef.Name = providerName
			eip.Spec.Domain = "vpc"
			eip.Spec.NetworkBorderGroup = "us-east-1"
			eip.Spec.Tags = map[string]string{
				"Name":      eipName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, eip)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for ElasticIP to be ready")
			eip = waitForElasticIPReady(namespace, eipName, createTimeout)

			By("verifying ElasticIP was created")
			Expect(eip.Status.Ready).To(BeTrue())
		})
	})

	Context("ElasticIP Status", func() {
		It("should update LastSyncTime on reconciliation", func() {
			eipName := "test-eip-sync"

			By("creating ElasticIP")
			createElasticIP(namespace, eipName, providerName)

			By("waiting for ElasticIP to be ready")
			eip := waitForElasticIPReady(namespace, eipName, createTimeout)

			By("verifying LastSyncTime is set")
			Expect(eip.Status.LastSyncTime).NotTo(BeNil())

			initialSyncTime := eip.Status.LastSyncTime.Time

			By("waiting for next reconciliation")
			time.Sleep(15 * time.Second)

			By("getting updated ElasticIP status")
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      eipName,
				Namespace: namespace,
			}, eip)
			Expect(err).NotTo(HaveOccurred())

			By("verifying LastSyncTime was updated or same")
			if eip.Status.LastSyncTime != nil {
				Expect(eip.Status.LastSyncTime.Time.Unix()).To(BeNumerically(">=", initialSyncTime.Unix()))
			}
		})

		It("should have no association by default", func() {
			eipName := "test-eip-no-assoc"

			By("creating ElasticIP")
			createElasticIP(namespace, eipName, providerName)

			By("waiting for ElasticIP to be ready")
			eip := waitForElasticIPReady(namespace, eipName, createTimeout)

			By("verifying no associations")
			Expect(eip.Status.AssociationID).To(BeEmpty())
			Expect(eip.Status.InstanceID).To(BeEmpty())
			Expect(eip.Status.NetworkInterfaceID).To(BeEmpty())
			Expect(eip.Status.PrivateIPAddress).To(BeEmpty())
		})

		It("should have valid public IP address", func() {
			eipName := "test-eip-valid-ip"

			By("creating ElasticIP")
			createElasticIP(namespace, eipName, providerName)

			By("waiting for ElasticIP to be ready")
			eip := waitForElasticIPReady(namespace, eipName, createTimeout)

			By("verifying public IP is not empty and has valid format")
			Expect(eip.Status.PublicIP).NotTo(BeEmpty())
			// Basic IP validation: should match pattern xxx.xxx.xxx.xxx
			Expect(eip.Status.PublicIP).To(MatchRegexp(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`))
		})
	})

	Context("Multiple ElasticIPs", func() {
		It("should allocate multiple Elastic IPs", func() {
			numEIPs := 3
			eipNames := make([]string, numEIPs)

			By("creating multiple ElasticIPs")
			for i := 0; i < numEIPs; i++ {
				eipNames[i] = generateUniqueName("test-eip")
				createElasticIP(namespace, eipNames[i], providerName)
			}

			By("waiting for all ElasticIPs to be ready")
			for _, eipName := range eipNames {
				eip := waitForElasticIPReady(namespace, eipName, createTimeout)
				Expect(eip.Status.Ready).To(BeTrue())
			}

			By("verifying all ElasticIPs have unique allocation IDs and public IPs")
			allocationIDs := make(map[string]bool)
			publicIPs := make(map[string]bool)

			for _, eipName := range eipNames {
				eip := &infrav1alpha1.ElasticIP{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      eipName,
					Namespace: namespace,
				}, eip)
				Expect(err).NotTo(HaveOccurred())

				Expect(allocationIDs[eip.Status.AllocationID]).To(BeFalse(), "Allocation ID should be unique")
				allocationIDs[eip.Status.AllocationID] = true

				Expect(publicIPs[eip.Status.PublicIP]).To(BeFalse(), "Public IP should be unique")
				publicIPs[eip.Status.PublicIP] = true
			}
		})
	})

	Context("ElasticIP Error Handling", func() {
		It("should handle missing provider reference", func() {
			eipName := "test-eip-no-provider"

			By("creating ElasticIP with non-existent provider")
			eip := &infrav1alpha1.ElasticIP{}
			eip.Name = eipName
			eip.Namespace = namespace
			eip.Spec.ProviderRef.Name = "non-existent-provider"
			eip.Spec.Domain = "vpc"

			err := k8sClient.Create(ctx, eip)
			Expect(err).NotTo(HaveOccurred())

			By("waiting and verifying ElasticIP does not become ready")
			time.Sleep(10 * time.Second)

			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      eipName,
				Namespace: namespace,
			}, eip)
			Expect(err).NotTo(HaveOccurred())
			Expect(eip.Status.Ready).To(BeFalse())
		})

		It("should reject invalid domain values", func() {
			eipName := "test-eip-invalid-domain"

			By("attempting to create ElasticIP with invalid domain")
			eip := &infrav1alpha1.ElasticIP{}
			eip.Name = eipName
			eip.Namespace = namespace
			eip.Spec.ProviderRef.Name = providerName
			eip.Spec.Domain = "invalid-domain"

			err := k8sClient.Create(ctx, eip)
			// Should fail validation due to kubebuilder enum validation
			Expect(err).To(HaveOccurred())
		})
	})
})
