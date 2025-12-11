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

var route53recordsetlog = logf.Log.WithName("route53recordset-resource")

func (r *Route53RecordSet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-aws-infra-operator-io-v1alpha1-route53recordset,mutating=false,failurePolicy=fail,sideEffects=None,groups=aws-infra-operator.runner.codes,resources=route53recordsets,verbs=create;update,versions=v1alpha1,name=vroute53recordset.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Route53RecordSet{}

func (r *Route53RecordSet) ValidateCreate() (admission.Warnings, error) {
	route53recordsetlog.Info("validate create", "name", r.Name)
	return r.validateRoute53RecordSet()
}

func (r *Route53RecordSet) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	route53recordsetlog.Info("validate update", "name", r.Name)

	oldRecord := old.(*Route53RecordSet)

	// Campos imutáveis
	if r.Spec.HostedZoneID != oldRecord.Spec.HostedZoneID {
		return nil, fmt.Errorf("spec.hostedZoneID is immutable")
	}
	if r.Spec.Name != oldRecord.Spec.Name {
		return nil, fmt.Errorf("spec.name is immutable")
	}
	if r.Spec.Type != oldRecord.Spec.Type {
		return nil, fmt.Errorf("spec.type is immutable")
	}

	return r.validateRoute53RecordSet()
}

func (r *Route53RecordSet) ValidateDelete() (admission.Warnings, error) {
	route53recordsetlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *Route53RecordSet) validateRoute53RecordSet() (admission.Warnings, error) {
	var warnings admission.Warnings

	// 1. Validar ProviderRef
	if r.Spec.ProviderRef.Name == "" {
		return nil, fmt.Errorf("spec.providerRef.name is required")
	}

	// 2. Validar HostedZoneID format
	if !regexp.MustCompile(`^Z[0-9A-Z]+$`).MatchString(r.Spec.HostedZoneID) {
		return nil, fmt.Errorf("spec.hostedZoneID must start with 'Z' followed by alphanumeric characters")
	}

	// 3. Validar alias target vs resource records (mutuamente exclusivos)
	if err := r.validateAliasTarget(); err != nil {
		return nil, err
	}

	// 4. Validar record type específico
	if err := r.validateRecordTypeRules(); err != nil {
		return nil, err
	}

	// 5. Validar routing policy
	if err := r.validateRoutingPolicy(); err != nil {
		return nil, err
	}

	// Warnings
	if r.Spec.DeletionPolicy == "" {
		warnings = append(warnings, "spec.deletionPolicy not set, defaulting to 'Delete'")
	}

	// Warn se TTL muito baixo
	if r.Spec.TTL != nil && *r.Spec.TTL < 60 {
		warnings = append(warnings, "TTL less than 60 seconds may cause high DNS query volume")
	}

	return warnings, nil
}

func (r *Route53RecordSet) validateAliasTarget() error {
	hasAlias := r.Spec.AliasTarget != nil
	hasResourceRecords := len(r.Spec.ResourceRecords) > 0
	hasTTL := r.Spec.TTL != nil

	// Alias e resource records são mutuamente exclusivos
	if hasAlias && hasResourceRecords {
		return fmt.Errorf("cannot specify both aliasTarget and resourceRecords")
	}

	// Alias não pode ter TTL
	if hasAlias && hasTTL {
		return fmt.Errorf("cannot specify TTL with aliasTarget")
	}

	// Se alias, validar campos obrigatórios
	if hasAlias {
		if r.Spec.AliasTarget.DNSName == "" {
			return fmt.Errorf("aliasTarget.dnsName is required")
		}
		if r.Spec.AliasTarget.HostedZoneID == "" {
			return fmt.Errorf("aliasTarget.hostedZoneID is required")
		}
	}

	// Se não é alias, deve ter resource records e TTL
	if !hasAlias {
		if !hasResourceRecords {
			return fmt.Errorf("resourceRecords is required for non-alias records")
		}
		if !hasTTL {
			return fmt.Errorf("ttl is required for non-alias records")
		}
	}

	return nil
}

func (r *Route53RecordSet) validateRecordTypeRules() error {
	switch r.Spec.Type {
	case "CNAME":
		// CNAME não pode coexistir com outros tipos no mesmo nome
		// (validação limitada sem acesso ao cluster)
		if r.Spec.Name == r.Spec.HostedZoneID {
			return fmt.Errorf("CNAME record cannot be created at zone apex")
		}

	case "MX", "SRV":
		// Validar formato dos resource records
		if r.Spec.AliasTarget == nil {
			for _, record := range r.Spec.ResourceRecords {
				if !regexp.MustCompile(`^\d+\s+.+$`).MatchString(record) {
					return fmt.Errorf("%s record must have priority/weight followed by target", r.Spec.Type)
				}
			}
		}

	case "TXT", "SPF":
		// TXT records devem estar entre aspas se contiverem espaços
		if r.Spec.AliasTarget == nil {
			for _, record := range r.Spec.ResourceRecords {
				if len(record) > 255 {
					return fmt.Errorf("TXT record value cannot exceed 255 characters")
				}
			}
		}
	}

	return nil
}

func (r *Route53RecordSet) validateRoutingPolicy() error {
	hasWeight := r.Spec.Weight != nil
	hasRegion := r.Spec.Region != ""
	hasGeoLocation := r.Spec.GeoLocation != nil
	hasFailover := r.Spec.Failover != ""

	// Contar quantas políticas de roteamento estão ativas
	activePolicies := 0
	if hasWeight {
		activePolicies++
	}
	if hasRegion {
		activePolicies++
	}
	if hasGeoLocation {
		activePolicies++
	}
	if hasFailover {
		activePolicies++
	}

	// Apenas uma política de roteamento pode estar ativa
	if activePolicies > 1 {
		return fmt.Errorf("only one routing policy can be specified (weight, region, geolocation, or failover)")
	}

	// Se tem política de roteamento, setIdentifier é obrigatório
	if activePolicies > 0 && r.Spec.SetIdentifier == "" {
		return fmt.Errorf("setIdentifier is required when using routing policies")
	}

	// Validar valores específicos
	if hasWeight && (*r.Spec.Weight < 0 || *r.Spec.Weight > 255) {
		return fmt.Errorf("weight must be between 0 and 255")
	}

	if hasGeoLocation {
		if r.Spec.GeoLocation.ContinentCode == "" && r.Spec.GeoLocation.CountryCode == "" {
			return fmt.Errorf("geolocation must specify at least continentCode or countryCode")
		}
	}

	return nil
}
