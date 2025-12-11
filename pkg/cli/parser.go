package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Resource representa um recurso genérico estilo Kubernetes do YAML
type Resource struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   Metadata               `yaml:"metadata"`
	Spec       map[string]interface{} `yaml:"spec"`
	Status     map[string]interface{} `yaml:"status,omitempty"`
}

// Metadata representa os metadados do recurso
type Metadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// ParseFile faz o parse de um arquivo YAML contendo um ou mais recursos
func ParseFile(filename string) ([]Resource, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo %s: %w", filename, err)
	}
	return ParseYAML(data)
}

// ParseYAML faz o parse de conteúdo YAML contendo um ou mais recursos (separados por ---)
// Usa o decoder YAML que lida nativamente com múltiplos documentos
func ParseYAML(data []byte) ([]Resource, error) {
	var resources []Resource

	// Cria decoder que lê múltiplos documentos
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	for {
		var res Resource
		err := decoder.Decode(&res)

		if err == io.EOF {
			break
		}

		if err != nil {
			// Pula documentos inválidos (comentários, vazios)
			continue
		}

		// Só adiciona se tem kind e apiVersion
		if res.Kind != "" && res.APIVersion != "" {
			resources = append(resources, res)
		}
	}

	return resources, nil
}

// FilterByKind retorna recursos de um tipo específico
func FilterByKind(resources []Resource, kind string) []Resource {
	var filtered []Resource
	for _, r := range resources {
		if strings.EqualFold(r.Kind, kind) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// GetResourceID retorna um identificador único para um recurso
func GetResourceID(r Resource) string {
	if r.Metadata.Namespace != "" {
		return fmt.Sprintf("%s/%s/%s", r.Kind, r.Metadata.Namespace, r.Metadata.Name)
	}
	return fmt.Sprintf("%s/%s", r.Kind, r.Metadata.Name)
}

// OrderByDependency ordena recursos baseado em suas dependências
// VPC -> Subnet -> SecurityGroup -> EC2Instance, etc.
func OrderByDependency(resources []Resource) []Resource {
	priority := map[string]int{
		"AWSProvider":          0,
		"VPC":                  1,
		"InternetGateway":      2,
		"Subnet":               3,
		"ElasticIP":            4,
		"NATGateway":           5,
		"RouteTable":           6,
		"SecurityGroup":        7,
		"IAMRole":              8,
		"KMSKey":               9,
		"SecretsManagerSecret": 10,
		"S3Bucket":             11,
		"ECRRepository":        12,
		"RDSInstance":          13,
		"DynamoDBTable":        14,
		"ElastiCacheCluster":   15,
		"SQSQueue":             16,
		"SNSTopic":             17,
		"LambdaFunction":       18,
		"EC2Instance":          19,
		"EC2KeyPair":           19,
		"ALB":                  20,
		"NLB":                  20,
		"EKSCluster":           21,
		"ECSCluster":           21,
		"APIGateway":           22,
		"Certificate":          23,
		"CloudFront":           24,
		"Route53HostedZone":    25,
		"Route53RecordSet":     26,
		"ComputeStack":         100, // ComputeStack cria tudo, então menor prioridade
	}

	// Bubble sort simples por prioridade (listas de recursos são pequenas)
	sorted := make([]Resource, len(resources))
	copy(sorted, resources)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			p1 := priority[sorted[j].Kind]
			p2 := priority[sorted[j+1].Kind]
			if p1 > p2 {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}
