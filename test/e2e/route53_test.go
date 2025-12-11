package e2e_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

var _ = Describe("Route53 E2E Tests", func() {
	var (
		namespace    string
		providerName string
	)

	BeforeEach(func() {
		namespace = generateUniqueName("route53-test")
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

	Context("Route53 HostedZone Lifecycle", func() {
		It("should create a hosted zone successfully", func() {
			zoneName := "test-zone"
			domainName := "example-e2e.com."

			By("creating Route53HostedZone CR")
			zone := &infrav1alpha1.Route53HostedZone{}
			zone.Name = zoneName
			zone.Namespace = namespace
			zone.Spec.ProviderRef.Name = providerName
			zone.Spec.Name = domainName
			zone.Spec.Comment = "E2E test hosted zone"
			zone.Spec.Tags = map[string]string{
				"Name":      zoneName,
				"ManagedBy": "infra-operator-e2e",
			}

			err := k8sClient.Create(ctx, zone)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for HostedZone to be ready")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      zoneName,
					Namespace: namespace,
				}, zone)
				if err != nil {
					return false
				}
				return zone.Status.Ready && zone.Status.HostedZoneID != ""
			}, createTimeout, pollInterval).Should(BeTrue())

			By("verifying HostedZone status")
			Expect(zone.Status.Ready).To(BeTrue())
			Expect(zone.Status.HostedZoneID).NotTo(BeEmpty())
			Expect(zone.Status.NameServers).NotTo(BeEmpty())
		})

		It("should delete hosted zone correctly", func() {
			zoneName := "test-zone-delete"
			domainName := "delete-example.com."

			By("creating Route53HostedZone")
			zone := &infrav1alpha1.Route53HostedZone{}
			zone.Name = zoneName
			zone.Namespace = namespace
			zone.Spec.ProviderRef.Name = providerName
			zone.Spec.Name = domainName

			err := k8sClient.Create(ctx, zone)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for HostedZone to be ready")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      zoneName,
					Namespace: namespace,
				}, zone)
				return err == nil && zone.Status.Ready
			}, createTimeout, pollInterval).Should(BeTrue())

			By("deleting HostedZone CR")
			err = k8sClient.Delete(ctx, zone)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for HostedZone to be removed")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      zoneName,
					Namespace: namespace,
				}, zone)
				return err != nil
			}, deleteTimeout, pollInterval).Should(BeTrue())
		})
	})

	Context("Route53 RecordSet Lifecycle", func() {
		It("should create an A record successfully", func() {
			// First create a hosted zone
			zoneName := "test-zone-for-records"
			domainName := "records-example.com."

			By("creating hosted zone")
			zone := &infrav1alpha1.Route53HostedZone{}
			zone.Name = zoneName
			zone.Namespace = namespace
			zone.Spec.ProviderRef.Name = providerName
			zone.Spec.Name = domainName

			err := k8sClient.Create(ctx, zone)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for HostedZone to be ready")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      zoneName,
					Namespace: namespace,
				}, zone)
				return err == nil && zone.Status.Ready && zone.Status.HostedZoneID != ""
			}, createTimeout, pollInterval).Should(BeTrue())

			hostedZoneID := zone.Status.HostedZoneID

			By("creating A record")
			recordName := "test-a-record"
			record := &infrav1alpha1.Route53RecordSet{}
			record.Name = recordName
			record.Namespace = namespace
			record.Spec.ProviderRef.Name = providerName
			record.Spec.HostedZoneID = hostedZoneID
			record.Spec.Name = "www.records-example.com."
			record.Spec.Type = "A"
			record.Spec.TTL = 300
			record.Spec.ResourceRecords = []string{"192.0.2.1"}

			err = k8sClient.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for RecordSet to be ready")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      recordName,
					Namespace: namespace,
				}, record)
				return err == nil && record.Status.Ready
			}, createTimeout, pollInterval).Should(BeTrue())

			By("verifying RecordSet status")
			Expect(record.Status.Ready).To(BeTrue())
			Expect(record.Status.FQDN).To(Equal("www.records-example.com."))
		})

		It("should create a CNAME record successfully", func() {
			zoneName := "test-zone-cname"
			domainName := "cname-example.com."

			By("creating hosted zone")
			zone := &infrav1alpha1.Route53HostedZone{}
			zone.Name = zoneName
			zone.Namespace = namespace
			zone.Spec.ProviderRef.Name = providerName
			zone.Spec.Name = domainName

			err := k8sClient.Create(ctx, zone)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      zoneName,
					Namespace: namespace,
				}, zone)
				return err == nil && zone.Status.Ready && zone.Status.HostedZoneID != ""
			}, createTimeout, pollInterval).Should(BeTrue())

			By("creating CNAME record")
			recordName := "test-cname-record"
			record := &infrav1alpha1.Route53RecordSet{}
			record.Name = recordName
			record.Namespace = namespace
			record.Spec.ProviderRef.Name = providerName
			record.Spec.HostedZoneID = zone.Status.HostedZoneID
			record.Spec.Name = "alias.cname-example.com."
			record.Spec.Type = "CNAME"
			record.Spec.TTL = 300
			record.Spec.ResourceRecords = []string{"target.example.com."}

			err = k8sClient.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for CNAME RecordSet to be ready")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      recordName,
					Namespace: namespace,
				}, record)
				return err == nil && record.Status.Ready
			}, createTimeout, pollInterval).Should(BeTrue())
		})
	})

	Context("Route53 Private Hosted Zone", func() {
		It("should create a private hosted zone with VPC association", func() {
			Skip("Private hosted zone requires VPC - test in integration suite")
		})
	})
})
