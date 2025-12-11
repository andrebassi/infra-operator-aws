package controllers

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	awsclients "infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

// Número máximo de linhas do console output a armazenar no status
const maxConsoleOutputLines = 100

const ec2InstanceFinalizerName = "ec2instance.aws-infra-operator.runner.codes/finalizer"

// EC2InstanceReconciler reconciles an EC2Instance object
type EC2InstanceReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *awsclients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=ec2instances/finalizers,verbs=update

func (r *EC2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the EC2Instance
	ec2Instance := &infrav1alpha1.EC2Instance{}
	if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get EC2 use case
	ec2UseCase, err := r.AWSClientFactory.GetEC2UseCase(ctx, ec2Instance.Spec.ProviderRef, ec2Instance.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get EC2 use case")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Check if being deleted
	if !ec2Instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(ec2Instance, ec2InstanceFinalizerName) {
			// Convert to domain
			instance := mapper.CRToDomainEC2Instance(ec2Instance)

			// Delete instance
			if err := ec2UseCase.DeleteInstance(ctx, instance); err != nil {
				logger.Error(err, "Failed to delete EC2 instance")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(ec2Instance, ec2InstanceFinalizerName)
			if err := r.Update(ctx, ec2Instance); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(ec2Instance, ec2InstanceFinalizerName) {
		controllerutil.AddFinalizer(ec2Instance, ec2InstanceFinalizerName)
		if err := r.Update(ctx, ec2Instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert to domain
	instance := mapper.CRToDomainEC2Instance(ec2Instance)

	// Sync instance
	if err := ec2UseCase.SyncInstance(ctx, instance); err != nil {
		logger.Error(err, "Failed to sync EC2 instance")
		ec2Instance.Status.Ready = false
		if updateErr := r.Status().Update(ctx, ec2Instance); updateErr != nil {
			logger.Error(updateErr, "Failed to update EC2Instance status")
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Update status
	mapper.DomainToStatusEC2Instance(instance, ec2Instance)

	// Busca console output se habilitado e instância está running
	if ec2Instance.Spec.EnableConsoleOutput && ec2Instance.Status.InstanceID != "" {
		if err := r.fetchConsoleOutput(ctx, ec2Instance); err != nil {
			logger.Error(err, "Falha ao obter console output", "instanceID", ec2Instance.Status.InstanceID)
			// Não falha a reconciliação por causa do console output
		}
	}

	if err := r.Status().Update(ctx, ec2Instance); err != nil {
		logger.Error(err, "Failed to update EC2Instance status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled EC2Instance",
		"instanceName", ec2Instance.Spec.InstanceName,
		"instanceID", ec2Instance.Status.InstanceID,
		"state", ec2Instance.Status.InstanceState,
		"consoleOutputEnabled", ec2Instance.Spec.EnableConsoleOutput)

	// Requeue after 5 minutes
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// fetchConsoleOutput busca os logs do console da EC2 e armazena no status
func (r *EC2InstanceReconciler) fetchConsoleOutput(ctx context.Context, ec2Instance *infrav1alpha1.EC2Instance) error {
	logger := log.FromContext(ctx)

	ec2Repo, err := r.AWSClientFactory.GetEC2Repository(ctx, ec2Instance.Spec.ProviderRef, ec2Instance.Namespace)
	if err != nil {
		return err
	}

	consoleOutput, err := ec2Repo.GetConsoleOutput(ctx, ec2Instance.Status.InstanceID, maxConsoleOutputLines)
	if err != nil {
		return err
	}

	ec2Instance.Status.ConsoleOutput = consoleOutput.Output
	timestamp := metav1.NewTime(consoleOutput.Timestamp)
	ec2Instance.Status.ConsoleOutputTimestamp = &timestamp

	logger.V(1).Info("Console output atualizado",
		"instanceID", ec2Instance.Status.InstanceID,
		"outputLength", len(consoleOutput.Output))

	return nil
}

func (r *EC2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.EC2Instance{}).
		Complete(r)
}
