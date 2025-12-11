package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	ddbadapter "infra-operator/internal/adapters/aws/dynamodb"
	ddbusecase "infra-operator/internal/usecases/dynamodb"
	"infra-operator/pkg/clients"
	"infra-operator/pkg/mapper"
)

const dynamodbTableFinalizer = "aws-infra-operator.runner.codes/dynamodbtable-finalizer"

// DynamoDBTableReconciler reconciles a DynamoDBTable object
type DynamoDBTableReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *clients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=dynamodbtables,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=dynamodbtables/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=dynamodbtables/finalizers,verbs=update

func (r *DynamoDBTableReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Fetch DynamoDBTable CR
	tableCR := &infrav1alpha1.DynamoDBTable{}
	if err := r.Get(ctx, req.NamespacedName, tableCR); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Get AWS configuration from provider
	awsConfig, _, err := r.AWSClientFactory.GetAWSConfigFromProviderRef(
		ctx,
		tableCR.Namespace,
		tableCR.Spec.ProviderRef,
	)
	if err != nil {
		logger.Error(err, "Failed to get AWS config from provider")
		return r.updateStatus(ctx, tableCR, false, fmt.Sprintf("Provider error: %v", err))
	}

	// 3. Create DynamoDB Repository (Adapter)
	ddbRepo := ddbadapter.NewRepository(awsConfig)

	// 4. Create Use Case with injected repository
	ddbUseCase := ddbusecase.NewTableUseCase(ddbRepo)

	// 5. Convert CR to Domain model
	domainTable := mapper.CRToDomainTable(tableCR)

	// 6. Handle deletion
	if !tableCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(tableCR, dynamodbTableFinalizer) {
			// Execute deletion through use case
			if err := ddbUseCase.DeleteTable(ctx, domainTable); err != nil {
				logger.Error(err, "Failed to delete DynamoDB table")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(tableCR, dynamodbTableFinalizer)
			if err := r.Update(ctx, tableCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 7. Add finalizer if not present
	if !controllerutil.ContainsFinalizer(tableCR, dynamodbTableFinalizer) {
		controllerutil.AddFinalizer(tableCR, dynamodbTableFinalizer)
		if err := r.Update(ctx, tableCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 8. Execute business logic through use case (idempotent)
	if err := ddbUseCase.SyncTable(ctx, domainTable); err != nil {
		logger.Error(err, "Failed to sync DynamoDB table")
		return r.updateStatus(ctx, tableCR, false, fmt.Sprintf("Sync failed: %v", err))
	}

	// 9. Update CR status from domain model
	mapper.DomainTableToCRStatus(domainTable, tableCR)
	tableCR.Status.Ready = domainTable.IsReady()
	tableCR.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "TableReady",
			Message:            fmt.Sprintf("DynamoDB table is %s", domainTable.Status),
		},
	}

	if err := r.Status().Update(ctx, tableCR); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled DynamoDBTable",
		"table", domainTable.Name,
		"status", domainTable.Status)

	// Requeue after 5 minutes for drift detection
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// updateStatus updates the CR status
func (r *DynamoDBTableReconciler) updateStatus(ctx context.Context, tableCR *infrav1alpha1.DynamoDBTable, ready bool, message string) (ctrl.Result, error) {
	tableCR.Status.Ready = ready

	conditionStatus := metav1.ConditionTrue
	reason := "TableReady"
	if !ready {
		conditionStatus = metav1.ConditionFalse
		reason = "TableNotReady"
	}

	tableCR.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             conditionStatus,
			LastTransitionTime: metav1.Now(),
			Reason:             reason,
			Message:            message,
		},
	}

	if err := r.Status().Update(ctx, tableCR); err != nil {
		return ctrl.Result{}, err
	}

	if !ready {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *DynamoDBTableReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.DynamoDBTable{}).
		Complete(r)
}
