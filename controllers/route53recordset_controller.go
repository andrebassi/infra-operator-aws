package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	awspkg "infra-operator/pkg/aws"
)

const route53RecordSetFinalizer = "route53recordset.aws-infra-operator.runner.codes/finalizer"

// Route53RecordSetReconciler reconciles a Route53RecordSet object
type Route53RecordSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=route53recordsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=route53recordsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=route53recordsets/finalizers,verbs=update

func (r *Route53RecordSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Route53RecordSet instance
	recordSet := &infrav1alpha1.Route53RecordSet{}
	if err := r.Get(ctx, req.NamespacedName, recordSet); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Route53RecordSet")
		return ctrl.Result{}, err
	}

	// Get AWS config from provider
	awsConfig, _, err := awspkg.GetAWSConfigFromProvider(ctx, r.Client, recordSet.Namespace, recordSet.Spec.ProviderRef)
	if err != nil {
		logger.Error(err, "Failed to get AWS config from provider")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Create Route53 client
	r53Client := route53.NewFromConfig(awsConfig)

	// Handle deletion
	if !recordSet.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, r53Client, recordSet)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(recordSet, route53RecordSetFinalizer) {
		controllerutil.AddFinalizer(recordSet, route53RecordSetFinalizer)
		if err := r.Update(ctx, recordSet); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Sync record set
	if err := r.syncRecordSet(ctx, r53Client, recordSet); err != nil {
		logger.Error(err, "Failed to sync Route53 record set")
		recordSet.Status.Ready = false
		if updateErr := r.Status().Update(ctx, recordSet); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Update status
	recordSet.Status.Ready = true
	now := metav1.Now()
	recordSet.Status.LastSyncTime = &now
	if err := r.Status().Update(ctx, recordSet); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *Route53RecordSetReconciler) syncRecordSet(ctx context.Context, r53Client *route53.Client, recordSet *infrav1alpha1.Route53RecordSet) error {
	logger := log.FromContext(ctx)

	// Validate hosted zone ID format
	hostedZoneID := recordSet.Spec.HostedZoneID
	if hostedZoneID[0] != '/' {
		hostedZoneID = "/hostedzone/" + hostedZoneID
	}

	// Check if record set exists
	existingRecord, err := r.getRecordSet(ctx, r53Client, recordSet)
	if err != nil {
		return fmt.Errorf("failed to check if record exists: %w", err)
	}

	// Build resource record set
	rrs := r.buildResourceRecordSet(recordSet)

	// Create or update record
	if existingRecord == nil {
		logger.Info("Creating Route53 record set", "name", recordSet.Spec.Name, "type", recordSet.Spec.Type)

		changeInput := &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: aws.String(hostedZoneID),
			ChangeBatch: &route53types.ChangeBatch{
				Changes: []route53types.Change{
					{
						Action:            route53types.ChangeActionCreate,
						ResourceRecordSet: rrs,
					},
				},
			},
		}

		output, err := r53Client.ChangeResourceRecordSets(ctx, changeInput)
		if err != nil {
			return fmt.Errorf("failed to create record set: %w", err)
		}

		recordSet.Status.ChangeID = aws.ToString(output.ChangeInfo.Id)
		recordSet.Status.ChangeStatus = string(output.ChangeInfo.Status)

		logger.Info("Successfully created Route53 record set", "changeID", recordSet.Status.ChangeID)
	} else {
		// Update if different
		if r.needsUpdate(existingRecord, rrs) {
			logger.Info("Updating Route53 record set", "name", recordSet.Spec.Name, "type", recordSet.Spec.Type)

			changeInput := &route53.ChangeResourceRecordSetsInput{
				HostedZoneId: aws.String(hostedZoneID),
				ChangeBatch: &route53types.ChangeBatch{
					Changes: []route53types.Change{
						{
							Action:            route53types.ChangeActionUpsert,
							ResourceRecordSet: rrs,
						},
					},
				},
			}

			output, err := r53Client.ChangeResourceRecordSets(ctx, changeInput)
			if err != nil {
				return fmt.Errorf("failed to update record set: %w", err)
			}

			recordSet.Status.ChangeID = aws.ToString(output.ChangeInfo.Id)
			recordSet.Status.ChangeStatus = string(output.ChangeInfo.Status)

			logger.Info("Successfully updated Route53 record set", "changeID", recordSet.Status.ChangeID)
		}
	}

	return nil
}

func (r *Route53RecordSetReconciler) getRecordSet(ctx context.Context, r53Client *route53.Client, recordSet *infrav1alpha1.Route53RecordSet) (*route53types.ResourceRecordSet, error) {
	hostedZoneID := recordSet.Spec.HostedZoneID
	if hostedZoneID[0] != '/' {
		hostedZoneID = "/hostedzone/" + hostedZoneID
	}

	// Ensure name ends with dot
	name := recordSet.Spec.Name
	if name[len(name)-1] != '.' {
		name = name + "."
	}

	listInput := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(hostedZoneID),
		StartRecordName: aws.String(name),
		StartRecordType: route53types.RRType(recordSet.Spec.Type),
		MaxItems:        aws.Int32(1),
	}

	output, err := r53Client.ListResourceRecordSets(ctx, listInput)
	if err != nil {
		return nil, err
	}

	for _, rrs := range output.ResourceRecordSets {
		if aws.ToString(rrs.Name) == name && string(rrs.Type) == recordSet.Spec.Type {
			// Check SetIdentifier if specified
			if recordSet.Spec.SetIdentifier != "" {
				if aws.ToString(rrs.SetIdentifier) == recordSet.Spec.SetIdentifier {
					return &rrs, nil
				}
			} else if rrs.SetIdentifier == nil {
				return &rrs, nil
			}
		}
	}

	return nil, nil
}

func (r *Route53RecordSetReconciler) buildResourceRecordSet(recordSet *infrav1alpha1.Route53RecordSet) *route53types.ResourceRecordSet {
	// Ensure name ends with dot
	name := recordSet.Spec.Name
	if name[len(name)-1] != '.' {
		name = name + "."
	}

	rrs := &route53types.ResourceRecordSet{
		Name: aws.String(name),
		Type: route53types.RRType(recordSet.Spec.Type),
	}

	// Set identifier if specified
	if recordSet.Spec.SetIdentifier != "" {
		rrs.SetIdentifier = aws.String(recordSet.Spec.SetIdentifier)
	}

	// Configure routing policy
	if recordSet.Spec.Weight != nil {
		rrs.Weight = recordSet.Spec.Weight
	}

	if recordSet.Spec.Region != "" {
		rrs.Region = route53types.ResourceRecordSetRegion(recordSet.Spec.Region)
	}

	if recordSet.Spec.GeoLocation != nil {
		rrs.GeoLocation = &route53types.GeoLocation{
			ContinentCode:   aws.String(recordSet.Spec.GeoLocation.ContinentCode),
			CountryCode:     aws.String(recordSet.Spec.GeoLocation.CountryCode),
			SubdivisionCode: aws.String(recordSet.Spec.GeoLocation.SubdivisionCode),
		}
	}

	if recordSet.Spec.Failover != "" {
		rrs.Failover = route53types.ResourceRecordSetFailover(recordSet.Spec.Failover)
	}

	if recordSet.Spec.MultiValueAnswer {
		rrs.MultiValueAnswer = aws.Bool(true)
	}

	if recordSet.Spec.HealthCheckID != "" {
		rrs.HealthCheckId = aws.String(recordSet.Spec.HealthCheckID)
	}

	// Configure alias or resource records
	if recordSet.Spec.AliasTarget != nil {
		rrs.AliasTarget = &route53types.AliasTarget{
			HostedZoneId:         aws.String(recordSet.Spec.AliasTarget.HostedZoneID),
			DNSName:              aws.String(recordSet.Spec.AliasTarget.DNSName),
			EvaluateTargetHealth: recordSet.Spec.AliasTarget.EvaluateTargetHealth,
		}
	} else {
		// Resource records
		if recordSet.Spec.TTL != nil {
			rrs.TTL = recordSet.Spec.TTL
		}

		resourceRecords := make([]route53types.ResourceRecord, len(recordSet.Spec.ResourceRecords))
		for i, value := range recordSet.Spec.ResourceRecords {
			resourceRecords[i] = route53types.ResourceRecord{
				Value: aws.String(value),
			}
		}
		rrs.ResourceRecords = resourceRecords
	}

	return rrs
}

func (r *Route53RecordSetReconciler) needsUpdate(existing, desired *route53types.ResourceRecordSet) bool {
	// Compare TTL
	if aws.ToInt64(existing.TTL) != aws.ToInt64(desired.TTL) {
		return true
	}

	// Compare resource records
	if len(existing.ResourceRecords) != len(desired.ResourceRecords) {
		return true
	}

	for i := range existing.ResourceRecords {
		if aws.ToString(existing.ResourceRecords[i].Value) != aws.ToString(desired.ResourceRecords[i].Value) {
			return true
		}
	}

	// Compare alias target
	if existing.AliasTarget != nil && desired.AliasTarget != nil {
		if aws.ToString(existing.AliasTarget.DNSName) != aws.ToString(desired.AliasTarget.DNSName) {
			return true
		}
		if aws.ToString(existing.AliasTarget.HostedZoneId) != aws.ToString(desired.AliasTarget.HostedZoneId) {
			return true
		}
	}

	// Compare routing policies
	if aws.ToInt64(existing.Weight) != aws.ToInt64(desired.Weight) {
		return true
	}

	if string(existing.Region) != string(desired.Region) {
		return true
	}

	if string(existing.Failover) != string(desired.Failover) {
		return true
	}

	return false
}

func (r *Route53RecordSetReconciler) handleDeletion(ctx context.Context, r53Client *route53.Client, recordSet *infrav1alpha1.Route53RecordSet) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(recordSet, route53RecordSetFinalizer) {
		// Check deletion policy
		if recordSet.Spec.DeletionPolicy != "Retain" {
			logger.Info("Deleting Route53 record set", "name", recordSet.Spec.Name, "type", recordSet.Spec.Type)

			hostedZoneID := recordSet.Spec.HostedZoneID
			if hostedZoneID[0] != '/' {
				hostedZoneID = "/hostedzone/" + hostedZoneID
			}

			// Get current record to delete
			existingRecord, err := r.getRecordSet(ctx, r53Client, recordSet)
			if err != nil {
				logger.Error(err, "Failed to get record for deletion")
				return ctrl.Result{RequeueAfter: time.Minute}, err
			}

			if existingRecord != nil {
				changeInput := &route53.ChangeResourceRecordSetsInput{
					HostedZoneId: aws.String(hostedZoneID),
					ChangeBatch: &route53types.ChangeBatch{
						Changes: []route53types.Change{
							{
								Action:            route53types.ChangeActionDelete,
								ResourceRecordSet: existingRecord,
							},
						},
					},
				}

				if _, err := r53Client.ChangeResourceRecordSets(ctx, changeInput); err != nil {
					logger.Error(err, "Failed to delete record set")
					return ctrl.Result{RequeueAfter: time.Minute}, err
				}

				logger.Info("Successfully deleted Route53 record set")
			}
		} else {
			logger.Info("Skipping record set deletion due to Retain policy")
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(recordSet, route53RecordSetFinalizer)
		if err := r.Update(ctx, recordSet); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Route53RecordSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.Route53RecordSet{}).
		Complete(r)
}
