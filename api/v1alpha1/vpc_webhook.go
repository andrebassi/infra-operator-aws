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

var vpclog = logf.Log.WithName("vpc-resource")

// SetupWebhookWithManager registra o webhook com o manager
func (r *VPC) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-aws-infra-operator-io-v1alpha1-vpc,mutating=false,failurePolicy=fail,sideEffects=None,groups=aws-infra-operator.runner.codes,resources=vpcs,verbs=create;update,versions=v1alpha1,name=vvpc.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &VPC{}

// ValidateCreate implementa webhook.Validator
func (r *VPC) ValidateCreate() (admission.Warnings, error) {
	vpclog.Info("validate create", "name", r.Name)

	return r.validateVPC()
}

// ValidateUpdate implementa webhook.Validator
func (r *VPC) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	vpclog.Info("validate update", "name", r.Name)

	// Verificar campos imutáveis
	oldVPC := old.(*VPC)
	if r.Spec.CidrBlock != oldVPC.Spec.CidrBlock {
		return nil, fmt.Errorf("spec.cidrBlock is immutable")
	}

	return r.validateVPC()
}

// ValidateDelete implementa webhook.Validator
func (r *VPC) ValidateDelete() (admission.Warnings, error) {
	vpclog.Info("validate delete", "name", r.Name)

	// Validações específicas de deleção (se houver)
	return nil, nil
}

// validateVPC contém validações comuns
func (r *VPC) validateVPC() (admission.Warnings, error) {
	var warnings admission.Warnings

	// 1. Validar CIDR block
	if err := r.validateCIDRBlock(); err != nil {
		return nil, err
	}

	// 2. Validar ProviderRef
	if r.Spec.ProviderRef.Name == "" {
		return nil, fmt.Errorf("spec.providerRef.name is required")
	}

	// 3. Validar Tags (não podem ter prefixo aws:)
	for key := range r.Spec.Tags {
		if regexp.MustCompile(`^aws:`).MatchString(key) {
			return nil, fmt.Errorf("tag keys cannot start with 'aws:': %s", key)
		}
	}

	// 4. Warnings (não bloqueiam)
	if r.Spec.DeletionPolicy == "" {
		warnings = append(warnings, "spec.deletionPolicy not set, defaulting to 'Delete'")
	}

	return warnings, nil
}

// validateCIDRBlock valida o CIDR block
func (r *VPC) validateCIDRBlock() error {
	_, ipNet, err := net.ParseCIDR(r.Spec.CidrBlock)
	if err != nil {
		return fmt.Errorf("invalid CIDR block: %v", err)
	}

	// Validar tamanho (/16 a /28)
	ones, bits := ipNet.Mask.Size()
	if bits != 32 || ones < 16 || ones > 28 {
		return fmt.Errorf("VPC CIDR block must be between /16 and /28")
	}

	return nil
}
