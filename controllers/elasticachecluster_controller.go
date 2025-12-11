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
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const elasticacheClusterFinalizerName = "elasticachecluster.aws-infra-operator.runner.codes/finalizer"

// ElastiCacheClusterReconciler reconciles a ElastiCacheCluster object
type ElastiCacheClusterReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=elasticacheclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=elasticacheclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=elasticacheclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (r *ElastiCacheClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ElastiCacheCluster instance
	elasticacheCluster := &infrav1alpha1.ElastiCacheCluster{}
	if err := r.Get(ctx, req.NamespacedName, elasticacheCluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get ElastiCache use case from provider
	elasticacheUseCase, err := r.AWSClientFactory.GetElastiCacheUseCase(ctx, elasticacheCluster.Spec.ProviderRef, elasticacheCluster.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get ElastiCache use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Check if the cluster is being deleted
	if !elasticacheCluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(elasticacheCluster, elasticacheClusterFinalizerName) {
			// Get auth token if needed
			var authToken string
			if elasticacheCluster.Spec.AuthTokenRef != nil {
				authToken, err = r.getSecretValue(ctx, elasticacheCluster.Namespace, elasticacheCluster.Spec.AuthTokenRef)
				if err != nil {
					logger.Error(err, "Failed to get auth token for deletion")
					// Continue with deletion anyway
				}
			}

			// Convert CR to domain model
			cluster := mapper.CRToDomainElastiCacheCluster(elasticacheCluster, authToken)

			// Delete the cluster
			if err := elasticacheUseCase.DeleteCluster(ctx, cluster); err != nil {
				logger.Error(err, "Failed to delete ElastiCache cluster")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(elasticacheCluster, elasticacheClusterFinalizerName)
			if err := r.Update(ctx, elasticacheCluster); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(elasticacheCluster, elasticacheClusterFinalizerName) {
		controllerutil.AddFinalizer(elasticacheCluster, elasticacheClusterFinalizerName)
		if err := r.Update(ctx, elasticacheCluster); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Get auth token if needed
	var authToken string
	if elasticacheCluster.Spec.AuthTokenRef != nil {
		authToken, err = r.getSecretValue(ctx, elasticacheCluster.Namespace, elasticacheCluster.Spec.AuthTokenRef)
		if err != nil {
			logger.Error(err, "Failed to get auth token")
			elasticacheCluster.Status.Ready = false
			r.Status().Update(ctx, elasticacheCluster)
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
		}
	}

	// Convert CR to domain model
	cluster := mapper.CRToDomainElastiCacheCluster(elasticacheCluster, authToken)

	// Sync the cluster
	if err := elasticacheUseCase.SyncCluster(ctx, cluster); err != nil {
		logger.Error(err, "Failed to sync ElastiCache cluster")
		elasticacheCluster.Status.Ready = false
		r.Status().Update(ctx, elasticacheCluster)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusElastiCacheCluster(cluster, elasticacheCluster)
	if err := r.Status().Update(ctx, elasticacheCluster); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue after 5 minutes to check cluster status
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *ElastiCacheClusterReconciler) getSecretValue(ctx context.Context, namespace string, selector *infrav1alpha1.SecretKeySelector) (string, error) {
	ns := selector.Namespace
	if ns == "" {
		ns = namespace
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      selector.Name,
		Namespace: ns,
	}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", ns, selector.Name, err)
	}

	value, ok := secret.Data[selector.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s/%s", selector.Key, ns, selector.Name)
	}

	return string(value), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ElastiCacheClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.ElastiCacheCluster{}).
		Complete(r)
}
