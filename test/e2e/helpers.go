package e2e_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

const (
	// Default timeouts for different operations
	createTimeout  = 3 * time.Minute
	updateTimeout  = 2 * time.Minute
	deleteTimeout  = 3 * time.Minute
	pollInterval   = 5 * time.Second
	shortTimeout   = 30 * time.Second
	mediumTimeout  = 2 * time.Minute
	longTimeout    = 5 * time.Minute
)

// TestResource is a generic interface for test resources
type TestResource interface {
	client.Object
	GetStatusReady() bool
}

// ========================================
// Namespace Helpers
// ========================================

// createNamespace creates a namespace for testing
func createNamespace(name string) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := k8sClient.Create(ctx, ns)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}

	return ns
}

// deleteNamespace deletes a namespace
func deleteNamespace(name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := k8sClient.Delete(ctx, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}

	// Wait for namespace deletion
	Eventually(func() bool {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, ns)
		return apierrors.IsNotFound(err)
	}, deleteTimeout, pollInterval).Should(BeTrue())
}

// ========================================
// AWSProvider Helpers
// ========================================

// createAWSProvider creates an AWSProvider for testing
func createAWSProvider(namespace, name string) *infrav1alpha1.AWSProvider {
	accessKey, secretKey, region := getLocalStackCredentials()

	// Create secret with credentials
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-credentials", name),
			Namespace: namespace,
		},
		StringData: map[string]string{
			"accessKeyID":     accessKey,
			"secretAccessKey": secretKey,
		},
	}
	Expect(k8sClient.Create(ctx, secret)).To(Succeed())

	// Create AWSProvider
	provider := &infrav1alpha1.AWSProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: infrav1alpha1.AWSProviderSpec{
			Region: region,
			CredentialsSecretRef: &infrav1alpha1.SecretReference{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		},
	}

	// If using LocalStack, set custom endpoint
	if isUsingLocalStack() {
		provider.Spec.Endpoint = getAWSEndpoint()
	}

	Expect(k8sClient.Create(ctx, provider)).To(Succeed())

	return provider
}

// waitForProviderReady waits for an AWSProvider to be ready
func waitForProviderReady(namespace, name string, timeout time.Duration) {
	Eventually(func() bool {
		provider := &infrav1alpha1.AWSProvider{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, provider)
		if err != nil {
			return false
		}
		return provider.Status.Ready
	}, timeout, pollInterval).Should(BeTrue())
}

// ========================================
// VPC Helpers
// ========================================

// createVPC creates a VPC resource
func createVPC(namespace, name, providerName, cidrBlock string) *infrav1alpha1.VPC {
	vpc := &infrav1alpha1.VPC{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: infrav1alpha1.VPCSpec{
			ProviderRef: infrav1alpha1.ProviderReference{
				Name: providerName,
			},
			CidrBlock:          cidrBlock,
			EnableDnsSupport:   true,
			EnableDnsHostnames: true,
			Tags: map[string]string{
				"Name":        name,
				"Environment": "e2e-test",
				"ManagedBy":   "infra-operator-e2e",
			},
		},
	}

	Expect(k8sClient.Create(ctx, vpc)).To(Succeed())
	return vpc
}

// waitForVPCReady waits for a VPC to be ready
func waitForVPCReady(namespace, name string, timeout time.Duration) *infrav1alpha1.VPC {
	vpc := &infrav1alpha1.VPC{}
	Eventually(func() bool {
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, vpc)
		if err != nil {
			return false
		}
		return vpc.Status.Ready && vpc.Status.VpcID != ""
	}, timeout, pollInterval).Should(BeTrue())

	return vpc
}

// deleteVPC deletes a VPC resource
func deleteVPC(namespace, name string) {
	vpc := &infrav1alpha1.VPC{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := k8sClient.Delete(ctx, vpc)
	if err != nil && !apierrors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

// waitForVPCDeleted waits for a VPC to be deleted
func waitForVPCDeleted(namespace, name string, timeout time.Duration) {
	Eventually(func() bool {
		vpc := &infrav1alpha1.VPC{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, vpc)
		return apierrors.IsNotFound(err)
	}, timeout, pollInterval).Should(BeTrue())
}

// ========================================
// S3Bucket Helpers
// ========================================

// createS3Bucket creates an S3Bucket resource
func createS3Bucket(namespace, name, providerName, bucketName string) *infrav1alpha1.S3Bucket {
	bucket := &infrav1alpha1.S3Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: infrav1alpha1.S3BucketSpec{
			ProviderRef: infrav1alpha1.ProviderReference{
				Name: providerName,
			},
			BucketName: bucketName,
			Tags: map[string]string{
				"Name":        bucketName,
				"Environment": "e2e-test",
				"ManagedBy":   "infra-operator-e2e",
			},
		},
	}

	Expect(k8sClient.Create(ctx, bucket)).To(Succeed())
	return bucket
}

// waitForS3BucketReady waits for an S3Bucket to be ready
func waitForS3BucketReady(namespace, name string, timeout time.Duration) *infrav1alpha1.S3Bucket {
	bucket := &infrav1alpha1.S3Bucket{}
	Eventually(func() bool {
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, bucket)
		if err != nil {
			return false
		}
		return bucket.Status.Ready && bucket.Status.BucketName != ""
	}, timeout, pollInterval).Should(BeTrue())

	return bucket
}

// deleteS3Bucket deletes an S3Bucket resource
func deleteS3Bucket(namespace, name string) {
	bucket := &infrav1alpha1.S3Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := k8sClient.Delete(ctx, bucket)
	if err != nil && !apierrors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

// waitForS3BucketDeleted waits for an S3Bucket to be deleted
func waitForS3BucketDeleted(namespace, name string, timeout time.Duration) {
	Eventually(func() bool {
		bucket := &infrav1alpha1.S3Bucket{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, bucket)
		return apierrors.IsNotFound(err)
	}, timeout, pollInterval).Should(BeTrue())
}

// ========================================
// ElasticIP Helpers
// ========================================

// createElasticIP creates an ElasticIP resource
func createElasticIP(namespace, name, providerName string) *infrav1alpha1.ElasticIP {
	eip := &infrav1alpha1.ElasticIP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: infrav1alpha1.ElasticIPSpec{
			ProviderRef: infrav1alpha1.ProviderReference{
				Name: providerName,
			},
			Domain: "vpc",
			Tags: map[string]string{
				"Name":        name,
				"Environment": "e2e-test",
				"ManagedBy":   "infra-operator-e2e",
			},
		},
	}

	Expect(k8sClient.Create(ctx, eip)).To(Succeed())
	return eip
}

// waitForElasticIPReady waits for an ElasticIP to be ready
func waitForElasticIPReady(namespace, name string, timeout time.Duration) *infrav1alpha1.ElasticIP {
	eip := &infrav1alpha1.ElasticIP{}
	Eventually(func() bool {
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, eip)
		if err != nil {
			return false
		}
		return eip.Status.Ready && eip.Status.AllocationID != "" && eip.Status.PublicIP != ""
	}, timeout, pollInterval).Should(BeTrue())

	return eip
}

// deleteElasticIP deletes an ElasticIP resource
func deleteElasticIP(namespace, name string) {
	eip := &infrav1alpha1.ElasticIP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := k8sClient.Delete(ctx, eip)
	if err != nil && !apierrors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

// waitForElasticIPDeleted waits for an ElasticIP to be deleted
func waitForElasticIPDeleted(namespace, name string, timeout time.Duration) {
	Eventually(func() bool {
		eip := &infrav1alpha1.ElasticIP{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, eip)
		return apierrors.IsNotFound(err)
	}, timeout, pollInterval).Should(BeTrue())
}

// ========================================
// Generic Helpers
// ========================================

// updateResource updates a resource and waits for the update to complete
func updateResource(obj client.Object) {
	Expect(k8sClient.Update(ctx, obj)).To(Succeed())
}

// patchResource patches a resource
func patchResource(obj client.Object, patch client.Patch) {
	Expect(k8sClient.Patch(ctx, obj, patch)).To(Succeed())
}

// getResource gets a resource
func getResource(namespace, name string, obj client.Object) error {
	return k8sClient.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, obj)
}

// waitForCondition waits for a condition to be true
func waitForCondition(condition func() bool, timeout time.Duration, message string) {
	Eventually(condition, timeout, pollInterval).Should(BeTrue(), message)
}

// generateUniqueName generates a unique name for resources
func generateUniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().Unix())
}

// cleanupResource ensures a resource is deleted
func cleanupResource(obj client.Object) {
	err := k8sClient.Delete(ctx, obj)
	if err != nil && !apierrors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

// removeFinalizer removes a finalizer from a resource
func removeFinalizer(obj client.Object, finalizer string) {
	finalizers := obj.GetFinalizers()
	newFinalizers := []string{}
	for _, f := range finalizers {
		if f != finalizer {
			newFinalizers = append(newFinalizers, f)
		}
	}
	obj.SetFinalizers(newFinalizers)
	Expect(k8sClient.Update(ctx, obj)).To(Succeed())
}
