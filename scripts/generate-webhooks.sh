#!/bin/bash

# Script para gerar validation webhooks para todos os CRDs do infra-operator

set -e

API_DIR="/Users/andrebassi/works/.solutions/operators/infra-operator/api/v1alpha1"

# Função para criar webhook básico
create_basic_webhook() {
    local resource=$1
    local Resource=$2
    local webhook_file="${API_DIR}/${resource}_webhook.go"

    if [ -f "$webhook_file" ]; then
        echo "✅ Webhook já existe: ${resource}_webhook.go"
        return
    fi

    cat > "$webhook_file" << EOF
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

var ${resource}log = logf.Log.WithName("${resource}-resource")

func (r *${Resource}) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-aws-infra-operator-io-v1alpha1-${resource},mutating=false,failurePolicy=fail,sideEffects=None,groups=aws-infra-operator.runner.codes,resources=${resource}s,verbs=create;update,versions=v1alpha1,name=v${resource}.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &${Resource}{}

func (r *${Resource}) ValidateCreate() (admission.Warnings, error) {
	${resource}log.Info("validate create", "name", r.Name)
	return r.validate${Resource}()
}

func (r *${Resource}) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	${resource}log.Info("validate update", "name", r.Name)
	return r.validate${Resource}()
}

func (r *${Resource}) ValidateDelete() (admission.Warnings, error) {
	${resource}log.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *${Resource}) validate${Resource}() (admission.Warnings, error) {
	var warnings admission.Warnings

	// 1. Validar ProviderRef
	if r.Spec.ProviderRef.Name == "" {
		return nil, fmt.Errorf("spec.providerRef.name is required")
	}

	// 2. Validar Tags (não podem ter prefixo aws:)
	for key := range r.Spec.Tags {
		if regexp.MustCompile(\`^aws:\`).MatchString(key) {
			return nil, fmt.Errorf("tag keys cannot start with 'aws:': %s", key)
		}
	}

	// 3. Warnings
	if r.Spec.DeletionPolicy == "" {
		warnings = append(warnings, "spec.deletionPolicy not set, defaulting to 'Delete'")
	}

	return warnings, nil
}
EOF

    echo "✅ Criado: ${resource}_webhook.go"
}

# Função para criar webhook test básico
create_basic_webhook_test() {
    local resource=$1
    local Resource=$2
    local test_file="${API_DIR}/${resource}_webhook_test.go"

    if [ -f "$test_file" ]; then
        echo "✅ Test já existe: ${resource}_webhook_test.go"
        return
    fi

    cat > "$test_file" << EOF
package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("${Resource} Webhook", func() {
	var obj *${Resource}

	BeforeEach(func() {
		obj = &${Resource}{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-${resource}",
				Namespace: "default",
			},
			Spec: ${Resource}Spec{
				ProviderRef: ProviderReference{Name: "test-provider"},
			},
		}
	})

	Context("ValidateCreate", func() {
		It("should accept valid ${Resource}", func() {
			warnings, err := obj.ValidateCreate()
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).NotTo(BeEmpty())
		})

		It("should reject empty ProviderRef", func() {
			obj.Spec.ProviderRef.Name = ""
			_, err := obj.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		It("should reject aws: prefix in tags", func() {
			obj.Spec.Tags = map[string]string{"aws:test": "value"}
			_, err := obj.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})
	})
})
EOF

    echo "✅ Criado: ${resource}_webhook_test.go"
}

# Recursos que já têm webhooks customizados (pular)
SKIP_RESOURCES="vpc s3bucket subnet route53recordset"

# Gerar webhooks para todos os recursos
echo "🚀 Gerando validation webhooks para todos os CRDs..."
echo ""

for types_file in ${API_DIR}/*_types.go; do
    filename=$(basename "$types_file")
    resource=$(echo "$filename" | sed 's/_types.go//')

    # Pular awsprovider e recursos já implementados
    if [ "$resource" == "awsprovider" ]; then
        continue
    fi

    if echo "$SKIP_RESOURCES" | grep -q "$resource"; then
        echo "⏭️  Pulando $resource (webhook customizado)"
        continue
    fi

    # Converter para PascalCase
    Resource=$(echo "$resource" | sed -E 's/(^|_)([a-z])/\U\2/g')

    echo "📝 Processando: $resource → $Resource"
    create_basic_webhook "$resource" "$Resource"
    create_basic_webhook_test "$resource" "$Resource"
    echo ""
done

echo ""
echo "✨ Webhooks gerados com sucesso!"
echo ""
echo "Próximos passos:"
echo "1. Revisar webhooks gerados em api/v1alpha1/*_webhook.go"
echo "2. Adicionar validações específicas para cada recurso"
echo "3. Executar: make generate-webhooks"
echo "4. Executar: make webhook-test"
EOF

chmod +x "$webhook_file"
