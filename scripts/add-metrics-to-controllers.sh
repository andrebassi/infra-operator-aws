#!/bin/bash
# Script to add metrics integration template to all controllers
# This is a GUIDE - review each controller individually

set -e

CONTROLLERS_DIR="/Users/andrebassi/works/.solutions/operators/infra-operator/controllers"
ADAPTERS_DIR="/Users/andrebassi/works/.solutions/operators/infra-operator/internal/adapters/aws"

echo "========================================"
echo "Infra Operator - Metrics Integration"
echo "========================================"
echo ""

# List of all controllers to update
CONTROLLERS=(
    "vpc_controller.go"
    "subnet_controller.go"
    "internetgateway_controller.go"
    "natgateway_controller.go"
    "securitygroup_controller.go"
    "routetable_controller.go"
    "alb_controller.go"
    "nlb_controller.go"
    "s3bucket_controller.go"
    "rdsinstance_controller.go"
    "dynamodbtable_controller.go"
    "sqsqueue_controller.go"
    "snstopic_controller.go"
    "apigateway_controller.go"
    "cloudfront_controller.go"
    "iamrole_controller.go"
    "secretsmanagersecret_controller.go"
    "kmskey_controller.go"
    "certificate_controller.go"
    "ecrrepository_controller.go"
    "ecscluster_controller.go"
    "ec2instance_controller.go"
    "lambdafunction_controller.go"
    "ekscluster_controller.go"
    "elasticachecluster_controller.go"
    "route53hostedzone_controller.go"
    "route53recordset_controller.go"
)

echo "Controllers to update: ${#CONTROLLERS[@]}"
echo ""

# Template for controller metrics integration
cat > /tmp/controller_metrics_template.txt << 'EOF'
// METRICS INTEGRATION TEMPLATE
// Add this import:
import (
    inframetrics "infra-operator/pkg/metrics"
)

// At the start of Reconcile():
func (r *ResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Iniciar recording de métricas de reconciliação
    metricsRecorder := inframetrics.NewReconcileMetricsRecorder("ResourceType")

    logger := log.FromContext(ctx)

    // Get resource
    resource := &infrav1alpha1.ResourceType{}
    if err := r.Get(ctx, req.NamespacedName, resource); err != nil {
        if client.IgnoreNotFound(err) == nil {
            statusRecorder := inframetrics.NewResourceStatusRecorder("ResourceType", req.Name)
            statusRecorder.Remove()
            return ctrl.Result{}, nil
        }
        metricsRecorder.RecordError(inframetrics.ErrorTypeGetFailed)
        return ctrl.Result{}, err
    }

    // Create status recorder
    statusRecorder := inframetrics.NewResourceStatusRecorder("ResourceType", resource.Name)

    // On provider error:
    if err != nil {
        metricsRecorder.RecordError(inframetrics.ErrorTypeProviderFailed)
        return ctrl.Result{}, err
    }

    // On deletion:
    if !resource.ObjectMeta.DeletionTimestamp.IsZero() {
        statusRecorder.SetDeleting()

        // Finalizer execution
        finalizerRecorder := inframetrics.NewFinalizerMetricsRecorder("ResourceType")

        if err := cleanup(); err != nil {
            finalizerRecorder.RecordError(inframetrics.ErrorTypeFinalizerFailed)
            metricsRecorder.RecordError(inframetrics.ErrorTypeFinalizerFailed)
            return ctrl.Result{}, err
        }

        finalizerRecorder.RecordSuccess()
        metricsRecorder.RecordSuccess()
        return ctrl.Result{}, nil
    }

    // On sync error:
    if err := sync(); err != nil {
        statusRecorder.SetNotReady()
        metricsRecorder.RecordError(inframetrics.ErrorTypeSyncFailed)
        return ctrl.Result{}, err
    }

    // On status update error:
    if err := r.Status().Update(ctx, resource); err != nil {
        metricsRecorder.RecordError(inframetrics.ErrorTypeStatusUpdateFailed)
        return ctrl.Result{}, err
    }

    // Update status metrics
    if resource.Status.Ready {
        statusRecorder.SetReady()
    } else {
        statusRecorder.SetNotReady()
    }

    // Record creation if new
    if resource.CreationTimestamp.Time.After(time.Now().Add(-1*time.Minute)) {
        statusRecorder.RecordCreation()
    }

    // Record success
    metricsRecorder.RecordSuccess()

    return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}
EOF

echo "✓ Created controller metrics template at /tmp/controller_metrics_template.txt"
echo ""

# Template for AWS adapter metrics
cat > /tmp/adapter_metrics_template.txt << 'EOF'
// AWS ADAPTER METRICS INTEGRATION TEMPLATE
// Add this import:
import (
    inframetrics "infra-operator/pkg/metrics"
)

// For each AWS API call:
func (r *Repository) SomeOperation(ctx context.Context, ...) error {
    // Start recording metrics
    metricsRecorder := inframetrics.NewAWSAPIMetricsRecorder(inframetrics.ServiceEC2, "OperationName")

    input := &ec2.OperationInput{...}

    output, err := r.client.Operation(ctx, input)
    if err != nil {
        metricsRecorder.RecordError(err)
        return fmt.Errorf("failed to ...: %w", err)
    }

    metricsRecorder.RecordSuccess()

    // Process output...
    return nil
}
EOF

echo "✓ Created adapter metrics template at /tmp/adapter_metrics_template.txt"
echo ""

# List AWS services and their constants
echo "AWS Service Constants:"
echo "  - inframetrics.ServiceEC2"
echo "  - inframetrics.ServiceS3"
echo "  - inframetrics.ServiceRDS"
echo "  - inframetrics.ServiceELBv2"
echo "  - inframetrics.ServiceIAM"
echo "  - inframetrics.ServiceSecretsManager"
echo "  - inframetrics.ServiceKMS"
echo "  - inframetrics.ServiceLambda"
echo "  - inframetrics.ServiceAPIGateway"
echo "  - inframetrics.ServiceCloudFront"
echo "  - inframetrics.ServiceRoute53"
echo "  - inframetrics.ServiceACM"
echo "  - inframetrics.ServiceDynamoDB"
echo "  - inframetrics.ServiceSQS"
echo "  - inframetrics.ServiceSNS"
echo "  - inframetrics.ServiceECR"
echo "  - inframetrics.ServiceECS"
echo "  - inframetrics.ServiceEKS"
echo "  - inframetrics.ServiceElastiCache"
echo ""

# Check which controllers already have metrics
echo "Checking for existing metrics integration..."
echo ""

UPDATED=0
NOT_UPDATED=0

for controller in "${CONTROLLERS[@]}"; do
    filepath="${CONTROLLERS_DIR}/${controller}"

    if [ ! -f "$filepath" ]; then
        echo "⚠ File not found: $controller"
        ((NOT_UPDATED++))
        continue
    fi

    if grep -q "inframetrics" "$filepath"; then
        echo "✓ $controller (already has metrics)"
        ((UPDATED++))
    else
        echo "✗ $controller (needs metrics)"
        ((NOT_UPDATED++))
    fi
done

echo ""
echo "========================================"
echo "Summary:"
echo "  Already integrated: $UPDATED"
echo "  Need integration:   $NOT_UPDATED"
echo "  Total controllers:  ${#CONTROLLERS[@]}"
echo "========================================"
echo ""

echo "Next Steps:"
echo ""
echo "1. Review the example implementation in:"
echo "   - controllers/elasticip_controller.go"
echo "   - internal/adapters/aws/elasticip/repository.go"
echo ""
echo "2. Follow the templates at:"
echo "   - /tmp/controller_metrics_template.txt (for controllers)"
echo "   - /tmp/adapter_metrics_template.txt (for AWS adapters)"
echo ""
echo "3. For each controller, add:"
echo "   a) Import: inframetrics \"infra-operator/pkg/metrics\""
echo "   b) ReconcileMetricsRecorder at start of Reconcile()"
echo "   c) ResourceStatusRecorder for status tracking"
echo "   d) FinalizerMetricsRecorder for cleanup"
echo "   e) Record success/error at each exit point"
echo ""
echo "4. For each AWS adapter repository, add:"
echo "   a) Import: inframetrics \"infra-operator/pkg/metrics\""
echo "   b) AWSAPIMetricsRecorder for each AWS API call"
echo "   c) RecordSuccess() or RecordError() after each call"
echo ""
echo "5. Test the metrics:"
echo "   kubectl port-forward -n infra-operator-system svc/controller-manager-metrics-service 8443:8443"
echo "   curl http://localhost:8443/metrics | grep infra_operator"
echo ""
echo "6. Deploy ServiceMonitor:"
echo "   kubectl apply -f config/prometheus/servicemonitor.yaml"
echo ""
echo "7. Import Grafana dashboard:"
echo "   config/grafana/dashboard.json"
echo ""

echo "✓ Metrics integration guide complete!"
