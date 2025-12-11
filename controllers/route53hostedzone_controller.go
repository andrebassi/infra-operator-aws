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

const route53HostedZoneFinalizer = "route53hostedzone.aws-infra-operator.runner.codes/finalizer"

// Route53HostedZoneReconciler reconciles a Route53HostedZone object
type Route53HostedZoneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=route53hostedzones,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=route53hostedzones/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=route53hostedzones/finalizers,verbs=update

func (r *Route53HostedZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Route53HostedZone instance
	hostedZone := &infrav1alpha1.Route53HostedZone{}
	if err := r.Get(ctx, req.NamespacedName, hostedZone); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Route53HostedZone")
		return ctrl.Result{}, err
	}

	// Get AWS config from provider
	awsConfig, provider, err := awspkg.GetAWSConfigFromProvider(ctx, r.Client, hostedZone.Namespace, hostedZone.Spec.ProviderRef)
	if err != nil {
		logger.Error(err, "Failed to get AWS config from provider")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Create Route53 client
	r53Client := route53.NewFromConfig(awsConfig)

	// Handle deletion
	if !hostedZone.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, r53Client, hostedZone)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(hostedZone, route53HostedZoneFinalizer) {
		controllerutil.AddFinalizer(hostedZone, route53HostedZoneFinalizer)
		if err := r.Update(ctx, hostedZone); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Sync hosted zone
	if err := r.syncHostedZone(ctx, r53Client, hostedZone, provider); err != nil {
		logger.Error(err, "Failed to sync Route53 hosted zone")
		hostedZone.Status.Ready = false
		if updateErr := r.Status().Update(ctx, hostedZone); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Update status
	hostedZone.Status.Ready = true
	now := metav1.Now()
	hostedZone.Status.LastSyncTime = &now
	if err := r.Status().Update(ctx, hostedZone); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *Route53HostedZoneReconciler) syncHostedZone(ctx context.Context, r53Client *route53.Client, hostedZone *infrav1alpha1.Route53HostedZone, provider *infrav1alpha1.AWSProvider) error {
	logger := log.FromContext(ctx)

	// Check if hosted zone exists
	var existingZone *route53types.HostedZone
	if hostedZone.Status.HostedZoneID != "" {
		getInput := &route53.GetHostedZoneInput{
			Id: aws.String(hostedZone.Status.HostedZoneID),
		}
		output, err := r53Client.GetHostedZone(ctx, getInput)
		if err == nil {
			existingZone = output.HostedZone
		}
	}

	// If zone doesn't exist, search by name
	if existingZone == nil {
		listInput := &route53.ListHostedZonesByNameInput{
			DNSName:  aws.String(hostedZone.Spec.Name),
			MaxItems: aws.Int32(1),
		}
		listOutput, err := r53Client.ListHostedZonesByName(ctx, listInput)
		if err != nil {
			return fmt.Errorf("failed to list hosted zones: %w", err)
		}

		// Check if we found a match
		if len(listOutput.HostedZones) > 0 {
			zone := listOutput.HostedZones[0]
			if aws.ToString(zone.Name) == hostedZone.Spec.Name+"." || aws.ToString(zone.Name) == hostedZone.Spec.Name {
				existingZone = &zone
			}
		}
	}

	// Create hosted zone if it doesn't exist
	if existingZone == nil {
		logger.Info("Creating Route53 hosted zone", "name", hostedZone.Spec.Name)

		createInput := &route53.CreateHostedZoneInput{
			Name:            aws.String(hostedZone.Spec.Name),
			CallerReference: aws.String(fmt.Sprintf("%s-%d", hostedZone.Name, time.Now().Unix())),
		}

		if hostedZone.Spec.Comment != "" {
			createInput.HostedZoneConfig = &route53types.HostedZoneConfig{
				Comment:     aws.String(hostedZone.Spec.Comment),
				PrivateZone: hostedZone.Spec.PrivateZone,
			}
		} else {
			createInput.HostedZoneConfig = &route53types.HostedZoneConfig{
				PrivateZone: hostedZone.Spec.PrivateZone,
			}
		}

		// Add VPC configuration for private zones
		if hostedZone.Spec.PrivateZone {
			if hostedZone.Spec.VPCId == "" || hostedZone.Spec.VPCRegion == "" {
				return fmt.Errorf("vpcId and vpcRegion are required for private hosted zones")
			}
			createInput.VPC = &route53types.VPC{
				VPCId:     aws.String(hostedZone.Spec.VPCId),
				VPCRegion: route53types.VPCRegion(hostedZone.Spec.VPCRegion),
			}
		}

		output, err := r53Client.CreateHostedZone(ctx, createInput)
		if err != nil {
			return fmt.Errorf("failed to create hosted zone: %w", err)
		}

		existingZone = output.HostedZone

		// Apply tags if specified
		if len(hostedZone.Spec.Tags) > 0 || len(provider.Spec.DefaultTags) > 0 {
			tags := make([]route53types.Tag, 0)

			// Add default tags from provider
			for k, v := range provider.Spec.DefaultTags {
				tags = append(tags, route53types.Tag{
					Key:   aws.String(k),
					Value: aws.String(v),
				})
			}

			// Add resource-specific tags (override defaults)
			for k, v := range hostedZone.Spec.Tags {
				tags = append(tags, route53types.Tag{
					Key:   aws.String(k),
					Value: aws.String(v),
				})
			}

			tagInput := &route53.ChangeTagsForResourceInput{
				ResourceType: route53types.TagResourceTypeHostedzone,
				ResourceId:   existingZone.Id,
				AddTags:      tags,
			}
			if _, err := r53Client.ChangeTagsForResource(ctx, tagInput); err != nil {
				logger.Error(err, "Failed to tag hosted zone")
			}
		}

		logger.Info("Successfully created Route53 hosted zone", "hostedZoneID", *existingZone.Id)
	}

	// Update status with hosted zone info
	hostedZone.Status.HostedZoneID = aws.ToString(existingZone.Id)
	hostedZone.Status.ResourceRecordSetCount = aws.ToInt64(existingZone.ResourceRecordSetCount)

	// Get name servers
	if existingZone.Id != nil {
		getInput := &route53.GetHostedZoneInput{
			Id: existingZone.Id,
		}
		output, err := r53Client.GetHostedZone(ctx, getInput)
		if err == nil && output.DelegationSet != nil {
			nameServers := make([]string, len(output.DelegationSet.NameServers))
			for i, ns := range output.DelegationSet.NameServers {
				nameServers[i] = ns
			}
			hostedZone.Status.NameServers = nameServers
		}
	}

	return nil
}

func (r *Route53HostedZoneReconciler) handleDeletion(ctx context.Context, r53Client *route53.Client, hostedZone *infrav1alpha1.Route53HostedZone) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(hostedZone, route53HostedZoneFinalizer) {
		// Check deletion policy
		if hostedZone.Spec.DeletionPolicy != "Retain" && hostedZone.Status.HostedZoneID != "" {
			logger.Info("Deleting Route53 hosted zone", "hostedZoneID", hostedZone.Status.HostedZoneID)

			deleteInput := &route53.DeleteHostedZoneInput{
				Id: aws.String(hostedZone.Status.HostedZoneID),
			}

			if _, err := r53Client.DeleteHostedZone(ctx, deleteInput); err != nil {
				logger.Error(err, "Failed to delete hosted zone")
				return ctrl.Result{RequeueAfter: time.Minute}, err
			}

			logger.Info("Successfully deleted Route53 hosted zone")
		} else {
			logger.Info("Skipping hosted zone deletion due to Retain policy")
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(hostedZone, route53HostedZoneFinalizer)
		if err := r.Update(ctx, hostedZone); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Route53HostedZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.Route53HostedZone{}).
		Complete(r)
}
