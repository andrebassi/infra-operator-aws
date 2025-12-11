// Package e2e_test provides end-to-end tests for the infra-operator.
//
// This test suite validates the operator against real AWS resources (or LocalStack),
// testing the complete lifecycle: create, reconcile, status updates, and deletion.
package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

var (
	// cfg is the rest config for the test environment
	cfg *rest.Config

	// k8sClient is the Kubernetes client for interacting with the cluster
	k8sClient client.Client

	// testEnv is the envtest environment
	testEnv *envtest.Environment

	// ctx is the context for all tests
	ctx context.Context

	// cancel cancels the context
	cancel context.CancelFunc

	// testNamespace is the namespace used for all tests
	testNamespace string

	// useLocalStack indicates if tests should use LocalStack (true) or real AWS (false)
	useLocalStack bool

	// awsEndpoint is the AWS endpoint (LocalStack or empty for real AWS)
	awsEndpoint string
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	// Check if we should use LocalStack
	useLocalStack = os.Getenv("USE_LOCALSTACK") != "false" // default to true
	if useLocalStack {
		awsEndpoint = os.Getenv("AWS_ENDPOINT_URL")
		if awsEndpoint == "" {
			awsEndpoint = "http://localhost:4566"
		}
		By(fmt.Sprintf("Using LocalStack at %s", awsEndpoint))
	} else {
		By("Using real AWS")
	}

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	By("adding custom resource schemes to the scheme")
	err = infrav1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	By("creating Kubernetes client")
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Create a unique namespace for this test run
	testNamespace = fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	By(fmt.Sprintf("creating test namespace: %s", testNamespace))
	// Namespace creation is handled per-test to avoid conflicts

	By("test environment setup complete")
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// Helper function to get the current test namespace
func getTestNamespace() string {
	if testNamespace == "" {
		testNamespace = fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	}
	return testNamespace
}

// Helper function to get AWS credentials for LocalStack
func getLocalStackCredentials() (string, string, string) {
	if useLocalStack {
		return "test", "test", "us-east-1"
	}
	// For real AWS, use environment variables or default credentials
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	return accessKey, secretKey, region
}

// Helper function to get AWS endpoint URL
func getAWSEndpoint() string {
	return awsEndpoint
}

// Helper function to check if using LocalStack
func isUsingLocalStack() bool {
	return useLocalStack
}

// withTimeout wraps a function with a timeout context
func withTimeout(timeout time.Duration, fn func(context.Context)) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	fn(ctx)
}

// logTestInfo logs information about the current test
func logTestInfo(message string, keysAndValues ...interface{}) {
	logger := ctrl.Log.WithName("e2e-test")
	logger.Info(message, keysAndValues...)
}
