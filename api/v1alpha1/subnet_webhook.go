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

var subnetlog = logf.Log.WithName("subnet-resource")

func (r *Subnet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-aws-infra-operator-io-v1alpha1-subnet,mutating=false,failurePolicy=fail,sideEffects=None,groups=aws-infra-operator.runner.codes,resources=subnets,verbs=create;update,versions=v1alpha1,name=vsubnet.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Subnet{}

func (r *Subnet) ValidateCreate() (admission.Warnings, error) {
	subnetlog.Info("validate create", "name", r.Name)
	return r.validateSubnet()
}

func (r *Subnet) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	subnetlog.Info("validate update", "name", r.Name)

	oldSubnet := old.(*Subnet)

	// Campos imutáveis
	if r.Spec.CidrBlock != oldSubnet.Spec.CidrBlock {
		return nil, fmt.Errorf("spec.cidrBlock is immutable")
	}
	if r.Spec.VpcID != oldSubnet.Spec.VpcID {
		return nil, fmt.Errorf("spec.vpcID is immutable")
	}
	if r.Spec.AvailabilityZone != oldSubnet.Spec.AvailabilityZone {
		return nil, fmt.Errorf("spec.availabilityZone is immutable")
	}

	return r.validateSubnet()
}

func (r *Subnet) ValidateDelete() (admission.Warnings, error) {
	subnetlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *Subnet) validateSubnet() (admission.Warnings, error) {
	var warnings admission.Warnings

	// 1. Validar CIDR block
	if err := r.validateCIDRBlock(); err != nil {
		return nil, err
	}

	// 2. Validar ProviderRef
	if r.Spec.ProviderRef.Name == "" {
		return nil, fmt.Errorf("spec.providerRef.name is required")
	}

	// 3. Validar VPC ID format
	if r.Spec.VpcID != "" {
		if !regexp.MustCompile(`^vpc-[0-9a-f]{8,17}$`).MatchString(r.Spec.VpcID) {
			return nil, fmt.Errorf("spec.vpcID must be in format 'vpc-xxxxxxxxx'")
		}
	}

	// 4. Validar Tags
	for key := range r.Spec.Tags {
		if regexp.MustCompile(`^aws:`).MatchString(key) {
			return nil, fmt.Errorf("tag keys cannot start with 'aws:': %s", key)
		}
	}

	// Warnings
	if r.Spec.DeletionPolicy == "" {
		warnings = append(warnings, "spec.deletionPolicy not set, defaulting to 'Delete'")
	}

	return warnings, nil
}

func (r *Subnet) validateCIDRBlock() error {
	_, ipNet, err := net.ParseCIDR(r.Spec.CidrBlock)
	if err != nil {
		return fmt.Errorf("invalid CIDR block: %v", err)
	}

	// Subnet deve ser /16 a /28
	ones, bits := ipNet.Mask.Size()
	if bits != 32 || ones < 16 || ones > 28 {
		return fmt.Errorf("subnet CIDR block must be between /16 and /28")
	}

	return nil
}
