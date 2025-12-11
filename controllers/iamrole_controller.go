package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	awsclients "infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const iamRoleFinalizerName = "iamrole.aws-infra-operator.runner.codes/finalizer"

// IAMRoleReconciler reconciles an IAMRole object
type IAMRoleReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *awsclients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=iamroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=iamroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=iamroles/finalizers,verbs=update

func (r *IAMRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the IAMRole instance
	iamRole := &infrav1alpha1.IAMRole{}
	if err := r.Get(ctx, req.NamespacedName, iamRole); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get IAM use case
	iamUseCase, err := r.AWSClientFactory.GetIAMUseCase(ctx, iamRole.Spec.ProviderRef, iamRole.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get IAM use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Check if the resource is being deleted
	if !iamRole.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(iamRole, iamRoleFinalizerName) {
			// Convert CR to domain model
			role := mapper.CRToDomainIAMRole(iamRole)

			// Delete IAM role
			if err := iamUseCase.DeleteRole(ctx, role); err != nil {
				logger.Error(err, "Failed to delete IAM role")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(iamRole, iamRoleFinalizerName)
			if err := r.Update(ctx, iamRole); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(iamRole, iamRoleFinalizerName) {
		controllerutil.AddFinalizer(iamRole, iamRoleFinalizerName)
		if err := r.Update(ctx, iamRole); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert CR to domain model
	role := mapper.CRToDomainIAMRole(iamRole)

	// Sync IAM role
	if err := iamUseCase.SyncRole(ctx, role); err != nil {
		logger.Error(err, "Failed to sync IAM role")
		iamRole.Status.Ready = false
		iamRole.Status.Message = err.Error()
		if updateErr := r.Status().Update(ctx, iamRole); updateErr != nil {
			logger.Error(updateErr, "Failed to update IAMRole status")
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusIAMRole(role, iamRole)
	if err := r.Status().Update(ctx, iamRole); err != nil {
		logger.Error(err, "Failed to update IAMRole status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled IAMRole",
		"roleName", iamRole.Spec.RoleName,
		"roleArn", iamRole.Status.RoleArn)

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *IAMRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.IAMRole{}).
		Complete(r)
}
