// Package v1alpha1 contém as definições de API para aws-infra-operator.runner.codes/v1alpha1.
//
// Este package define todos os Custom Resource Definitions (CRDs) para gerenciamento
// de recursos AWS através do Kubernetes.
package v1alpha1

import (
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var route53hostedzonelog = logf.Log.WithName("route53hostedzone-resource")

func (r *Route53HostedZone) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-aws-infra-operator-io-v1alpha1-route53hostedzone,mutating=false,failurePolicy=fail,sideEffects=None,groups=aws-infra-operator.runner.codes,resources=route53hostedzones,verbs=create;update,versions=v1alpha1,name=vroute53hostedzone.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Route53HostedZone{}

func (r *Route53HostedZone) ValidateCreate() (admission.Warnings, error) {
	route53hostedzonelog.Info("validate create", "name", r.Name)
	return r.validateRoute53HostedZone()
}

func (r *Route53HostedZone) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	route53hostedzonelog.Info("validate update", "name", r.Name)
	return r.validateRoute53HostedZone()
}

func (r *Route53HostedZone) ValidateDelete() (admission.Warnings, error) {
	route53hostedzonelog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *Route53HostedZone) validateRoute53HostedZone() (admission.Warnings, error) {
	var warnings admission.Warnings

	// 1. Validar ProviderRef
	if r.Spec.ProviderRef.Name == "" {
		return nil, fmt.Errorf("spec.providerRef.name is required")
	}

	// 2. Validar Tags (não podem ter prefixo aws:)
	for key := range r.Spec.Tags {
		if regexp.MustCompile(`^aws:`).MatchString(key) {
			return nil, fmt.Errorf("tag keys cannot start with 'aws:': %s", key)
		}
	}

	// 3. Warnings
	if r.Spec.DeletionPolicy == "" {
		warnings = append(warnings, "spec.deletionPolicy not set, defaulting to 'Delete'")
	}

	return warnings, nil
}
