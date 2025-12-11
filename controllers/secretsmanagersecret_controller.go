package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	awsclients "infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const secretsManagerSecretFinalizerName = "secretsmanagersecret.aws-infra-operator.runner.codes/finalizer"

type SecretsManagerSecretReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *awsclients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=secretsmanagersecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=secretsmanagersecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=secretsmanagersecrets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *SecretsManagerSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	smSecret := &infrav1alpha1.SecretsManagerSecret{}
	if err := r.Get(ctx, req.NamespacedName, smSecret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	smUseCase, err := r.AWSClientFactory.GetSecretsManagerUseCase(ctx, smSecret.Spec.ProviderRef, smSecret.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get Secrets Manager use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	if !smSecret.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(smSecret, secretsManagerSecretFinalizerName) {
			secretValue, err := r.getSecretValue(ctx, smSecret)
			if err != nil {
				logger.Error(err, "Failed to get secret value")
				return ctrl.Result{}, err
			}

			secret := mapper.CRToDomainSecretsManagerSecret(smSecret, secretValue)
			if err := smUseCase.DeleteSecret(ctx, secret); err != nil {
				logger.Error(err, "Failed to delete secret")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(smSecret, secretsManagerSecretFinalizerName)
			if err := r.Update(ctx, smSecret); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(smSecret, secretsManagerSecretFinalizerName) {
		controllerutil.AddFinalizer(smSecret, secretsManagerSecretFinalizerName)
		if err := r.Update(ctx, smSecret); err != nil {
			return ctrl.Result{}, err
		}
	}

	secretValue, err := r.getSecretValue(ctx, smSecret)
	if err != nil {
		logger.Error(err, "Failed to get secret value")
		smSecret.Status.Ready = false
		r.Status().Update(ctx, smSecret)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	secret := mapper.CRToDomainSecretsManagerSecret(smSecret, secretValue)
	if err := smUseCase.SyncSecret(ctx, secret); err != nil {
		logger.Error(err, "Failed to sync secret")
		smSecret.Status.Ready = false
		r.Status().Update(ctx, smSecret)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	mapper.DomainToStatusSecretsManagerSecret(secret, smSecret)
	if err := r.Status().Update(ctx, smSecret); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled SecretsManagerSecret", "secretName", smSecret.Spec.SecretName, "arn", smSecret.Status.ARN)
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *SecretsManagerSecretReconciler) getSecretValue(ctx context.Context, smSecret *infrav1alpha1.SecretsManagerSecret) (string, error) {
	if smSecret.Spec.SecretStringRef == nil {
		return "", fmt.Errorf("secretStringRef is required")
	}

	k8sSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      smSecret.Spec.SecretStringRef.Name,
		Namespace: smSecret.Namespace,
	}, k8sSecret); err != nil {
		return "", fmt.Errorf("failed to get Kubernetes secret: %w", err)
	}

	value, ok := k8sSecret.Data[smSecret.Spec.SecretStringRef.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s", smSecret.Spec.SecretStringRef.Key, smSecret.Spec.SecretStringRef.Name)
	}

	return string(value), nil
}

func (r *SecretsManagerSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.SecretsManagerSecret{}).
		Complete(r)
}
