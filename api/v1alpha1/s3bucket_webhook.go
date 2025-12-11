// Package v1alpha1 contém as definições de API para aws-infra-operator.runner.codes/v1alpha1.
//
// Este package define todos os Custom Resource Definitions (CRDs) para gerenciamento
// de recursos AWS através do Kubernetes.
package v1alpha1

import (
	"fmt"
	"net"
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var s3bucketlog = logf.Log.WithName("s3bucket-resource")

func (r *S3Bucket) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-aws-infra-operator-io-v1alpha1-s3bucket,mutating=false,failurePolicy=fail,sideEffects=None,groups=aws-infra-operator.runner.codes,resources=s3buckets,verbs=create;update,versions=v1alpha1,name=vs3bucket.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &S3Bucket{}

func (r *S3Bucket) ValidateCreate() (admission.Warnings, error) {
	s3bucketlog.Info("validate create", "name", r.Name)
	return r.validateS3Bucket()
}

func (r *S3Bucket) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	s3bucketlog.Info("validate update", "name", r.Name)

	// BucketName é imutável
	oldBucket := old.(*S3Bucket)
	if r.Spec.BucketName != oldBucket.Spec.BucketName {
		return nil, fmt.Errorf("spec.bucketName is immutable")
	}

	return r.validateS3Bucket()
}

func (r *S3Bucket) ValidateDelete() (admission.Warnings, error) {
	s3bucketlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *S3Bucket) validateS3Bucket() (admission.Warnings, error) {
	var warnings admission.Warnings

	// 1. Validar nome do bucket
	if err := r.validateBucketName(); err != nil {
		return nil, err
	}

	// 2. Validar ProviderRef
	if r.Spec.ProviderRef.Name == "" {
		return nil, fmt.Errorf("spec.providerRef.name is required")
	}

	// 3. Validar encryption se especificada
	if r.Spec.Encryption != nil {
		if err := r.validateEncryption(); err != nil {
			return nil, err
		}
	}

	// 4. Validar lifecycle rules
	for i, rule := range r.Spec.LifecycleRules {
		if err := r.validateLifecycleRule(rule, i); err != nil {
			return nil, err
		}
	}

	// 5. Validar Tags
	for key := range r.Spec.Tags {
		if regexp.MustCompile(`^aws:`).MatchString(key) {
			return nil, fmt.Errorf("tag keys cannot start with 'aws:': %s", key)
		}
	}

	// Warnings
	if r.Spec.DeletionPolicy == "" {
		warnings = append(warnings, "spec.deletionPolicy not set, defaulting to 'Delete'")
	}

	if r.Spec.Versioning == nil || !r.Spec.Versioning.Enabled {
		warnings = append(warnings, "versioning not enabled - data loss possible on accidental deletion")
	}

	return warnings, nil
}

func (r *S3Bucket) validateBucketName() error {
	name := r.Spec.BucketName

	// 3-63 caracteres
	if len(name) < 3 || len(name) > 63 {
		return fmt.Errorf("bucket name must be 3-63 characters, got %d", len(name))
	}

	// Lowercase, números, hífens, pontos
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`).MatchString(name) {
		return fmt.Errorf("invalid bucket name format: must start/end with lowercase letter or number, contain only lowercase letters, numbers, hyphens, and periods")
	}

	// Não pode ter .. consecutivos
	if regexp.MustCompile(`\.\.`).MatchString(name) {
		return fmt.Errorf("bucket name cannot contain consecutive periods")
	}

	// Não pode parecer com IP address
	if net.ParseIP(name) != nil {
		return fmt.Errorf("bucket name cannot be formatted as IP address")
	}

	// Não pode começar com xn--
	if regexp.MustCompile(`^xn--`).MatchString(name) {
		return fmt.Errorf("bucket name cannot start with 'xn--'")
	}

	// Não pode terminar com -s3alias
	if regexp.MustCompile(`-s3alias$`).MatchString(name) {
		return fmt.Errorf("bucket name cannot end with '-s3alias'")
	}

	return nil
}

func (r *S3Bucket) validateEncryption() error {
	if r.Spec.Encryption.Algorithm == "aws_kms" && r.Spec.Encryption.KMSKeyID == "" {
		return fmt.Errorf("spec.encryption.kmsKeyID is required when algorithm is 'aws_kms'")
	}
	return nil
}

func (r *S3Bucket) validateLifecycleRule(rule LifecycleRule, index int) error {
	if rule.ID == "" {
		return fmt.Errorf("spec.lifecycleRules[%d].id is required", index)
	}

	// Deve ter expiration ou transitions
	if rule.Expiration == nil && len(rule.Transitions) == 0 {
		return fmt.Errorf("spec.lifecycleRules[%d] must have either expiration or transitions", index)
	}

	// Validar transitions
	for j, transition := range rule.Transitions {
		if transition.Days < 0 {
			return fmt.Errorf("spec.lifecycleRules[%d].transitions[%d].days must be >= 0", index, j)
		}
	}

	return nil
}
