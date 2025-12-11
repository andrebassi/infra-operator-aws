package mapper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/lambda"
)

// CRToDomainFunction converts a LambdaFunction CR to domain model
func CRToDomainFunction(cr *infrav1alpha1.LambdaFunction) *lambda.Function {
	function := &lambda.Function{
		Name:           cr.Spec.FunctionName,
		Description:    cr.Spec.Description,
		Runtime:        cr.Spec.Runtime,
		Handler:        cr.Spec.Handler,
		Role:           cr.Spec.Role,
		Timeout:        cr.Spec.Timeout,
		MemorySize:     cr.Spec.MemorySize,
		Layers:         cr.Spec.Layers,
		Tags:           cr.Spec.Tags,
		DeletionPolicy: cr.Spec.DeletionPolicy,
	}

	// Map code
	function.Code = lambda.Code{
		ZipFile:         cr.Spec.Code.ZipFile,
		S3Bucket:        cr.Spec.Code.S3Bucket,
		S3Key:           cr.Spec.Code.S3Key,
		S3ObjectVersion: cr.Spec.Code.S3ObjectVersion,
		ImageUri:        cr.Spec.Code.ImageUri,
	}

	// Map environment variables
	if cr.Spec.Environment != nil {
		function.Environment = cr.Spec.Environment.Variables
	}

	// Map VPC configuration
	if cr.Spec.VpcConfig != nil {
		function.VpcConfig = &lambda.VpcConfig{
			SecurityGroupIds: cr.Spec.VpcConfig.SecurityGroupIds,
			SubnetIds:        cr.Spec.VpcConfig.SubnetIds,
		}
	}

	// Copy status fields if available
	if cr.Status.FunctionArn != "" {
		function.ARN = cr.Status.FunctionArn
	}
	function.State = cr.Status.State
	function.StateReason = cr.Status.StateReason
	function.Version = cr.Status.Version
	function.CodeSize = cr.Status.CodeSize

	// LastModified is a string in CR status, parse it if available
	if cr.Status.LastModified != "" {
		// Leave as zero time if parsing fails - not critical
	}

	return function
}

// DomainFunctionToStatus converts domain model to CR status
func DomainFunctionToStatus(function *lambda.Function) infrav1alpha1.LambdaFunctionStatus {
	status := infrav1alpha1.LambdaFunctionStatus{
		Ready:        function.IsActive(),
		FunctionArn:  function.ARN,
		Version:      function.Version,
		CodeSize:     function.CodeSize,
		State:        function.State,
		StateReason:  function.StateReason,
		LastSyncTime: metav1.Now(),
	}

	// Set last modified time
	if !function.LastModified.IsZero() {
		status.LastModified = metav1.NewTime(function.LastModified).String()
	}

	return status
}
