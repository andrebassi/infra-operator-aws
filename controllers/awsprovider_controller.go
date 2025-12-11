package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
)

// AWSProviderReconciler reconciles an AWSProvider object
type AWSProviderReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=awsproviders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=awsproviders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=awsproviders/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *AWSProviderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the AWSProvider instance
	provider := &infrav1alpha1.AWSProvider{}
	if err := r.Get(ctx, req.NamespacedName, provider); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Build AWS config
	awsConfig, err := r.buildAWSConfig(ctx, provider)
	if err != nil {
		logger.Error(err, "Failed to build AWS config")
		return r.updateStatus(ctx, provider, false, err.Error())
	}

	// Verify credentials using STS GetCallerIdentity
	stsClient := sts.NewFromConfig(awsConfig)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		logger.Error(err, "Failed to get caller identity")
		return r.updateStatus(ctx, provider, false, fmt.Sprintf("Authentication failed: %v", err))
	}

	// Update status with success
	provider.Status.Ready = true
	provider.Status.AccountID = aws.ToString(identity.Account)
	provider.Status.CallerIdentity = aws.ToString(identity.Arn)
	now := metav1.Now()
	provider.Status.LastAuthenticationTime = &now

	// Update conditions
	provider.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "AuthenticationSucceeded",
			Message:            fmt.Sprintf("Successfully authenticated as %s", aws.ToString(identity.Arn)),
		},
	}

	if err := r.Status().Update(ctx, provider); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled AWSProvider",
		"account", provider.Status.AccountID,
		"identity", provider.Status.CallerIdentity)

	// Requeue after 5 minutes to re-verify credentials
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *AWSProviderReconciler) buildAWSConfig(ctx context.Context, provider *infrav1alpha1.AWSProvider) (aws.Config, error) {
	var cfg aws.Config
	var err error

	// Base config with region
	configOptions := []func(*config.LoadOptions) error{
		config.WithRegion(provider.Spec.Region),
	}

	// If CredentialsSecret is provided (new preferred method)
	if provider.Spec.CredentialsSecret != nil {
		namespace := provider.Spec.CredentialsSecret.Namespace
		if namespace == "" {
			namespace = provider.Namespace
		}

		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      provider.Spec.CredentialsSecret.Name,
			Namespace: namespace,
		}, secret); err != nil {
			return cfg, fmt.Errorf("failed to get credentials secret: %w", err)
		}

		accessKey := string(secret.Data["AWS_ACCESS_KEY_ID"])
		secretKey := string(secret.Data["AWS_SECRET_ACCESS_KEY"])

		if accessKey == "" || secretKey == "" {
			return cfg, fmt.Errorf("AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY not found in secret")
		}

		configOptions = append(configOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	} else if provider.Spec.AccessKeyIDRef != nil && provider.Spec.SecretAccessKeyRef != nil {
		// If AccessKey/SecretKey are provided via secret references (deprecated)
		accessKey, err := r.getSecretValue(ctx, provider, provider.Spec.AccessKeyIDRef)
		if err != nil {
			return cfg, fmt.Errorf("failed to get access key: %w", err)
		}

		secretKey, err := r.getSecretValue(ctx, provider, provider.Spec.SecretAccessKeyRef)
		if err != nil {
			return cfg, fmt.Errorf("failed to get secret key: %w", err)
		}

		configOptions = append(configOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	// If RoleARN is provided, use it (IRSA or assume role)
	if provider.Spec.RoleARN != "" {
		// The AWS SDK will automatically use IRSA if running in EKS with proper service account
		// credentials are loaded automatically from pod service account
	}

	// Load config with all options
	cfg, err = config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return cfg, err
	}

	// If custom endpoint is provided (for LocalStack or custom AWS endpoint)
	if provider.Spec.Endpoint != "" {
		cfg.BaseEndpoint = aws.String(provider.Spec.Endpoint)
	}

	return cfg, nil
}

func (r *AWSProviderReconciler) getSecretValue(ctx context.Context, provider *infrav1alpha1.AWSProvider, selector *infrav1alpha1.SecretKeySelector) (string, error) {
	namespace := selector.Namespace
	if namespace == "" {
		namespace = provider.Namespace
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      selector.Name,
		Namespace: namespace,
	}, secret); err != nil {
		return "", err
	}

	value, ok := secret.Data[selector.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s", selector.Key, selector.Name)
	}

	return string(value), nil
}

func (r *AWSProviderReconciler) updateStatus(ctx context.Context, provider *infrav1alpha1.AWSProvider, ready bool, message string) (ctrl.Result, error) {
	provider.Status.Ready = ready

	conditionStatus := metav1.ConditionTrue
	reason := "AuthenticationSucceeded"
	if !ready {
		conditionStatus = metav1.ConditionFalse
		reason = "AuthenticationFailed"
	}

	provider.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             conditionStatus,
			LastTransitionTime: metav1.Now(),
			Reason:             reason,
			Message:            message,
		},
	}

	if err := r.Status().Update(ctx, provider); err != nil {
		return ctrl.Result{}, err
	}

	// Retry after 1 minute on failure
	if !ready {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AWSProviderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.AWSProvider{}).
		Complete(r)
}
