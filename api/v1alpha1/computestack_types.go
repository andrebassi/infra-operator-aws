/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComputeStackSpec define o estado desejado do ComputeStack
// Um ComputeStack cria toda a infraestrutura de rede necessária:
// VPC, Subnets, Internet Gateway, NAT Gateway, Route Tables e Security Groups
// Todos os campos são opcionais se você passar os IDs de recursos existentes
type ComputeStackSpec struct {
	// ProviderRef referencia o AWSProvider para autenticação
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// ===========================================================================
	// Campos para usar recursos EXISTENTES (se passar ID, não cria novo)
	// ===========================================================================

	// ExistingVpcID é o ID de uma VPC existente para usar
	// Se especificado, não cria nova VPC e ignora vpcCIDR/vpcName
	// +optional
	ExistingVpcID string `json:"existingVpcID,omitempty"`

	// ExistingSubnetIDs são IDs de subnets existentes para usar
	// Se especificado, não cria novas subnets e ignora publicSubnets/privateSubnets
	// +optional
	ExistingSubnetIDs []string `json:"existingSubnetIDs,omitempty"`

	// ExistingInternetGatewayID é o ID de um Internet Gateway existente
	// Se especificado, não cria novo IGW
	// +optional
	ExistingInternetGatewayID string `json:"existingInternetGatewayID,omitempty"`

	// ExistingNATGatewayIDs são IDs de NAT Gateways existentes
	// Se especificado, não cria novos NAT Gateways
	// +optional
	ExistingNATGatewayIDs []string `json:"existingNATGatewayIDs,omitempty"`

	// ExistingRouteTableIDs são IDs de Route Tables existentes
	// Se especificado, não cria novas Route Tables
	// +optional
	ExistingRouteTableIDs []string `json:"existingRouteTableIDs,omitempty"`

	// ExistingSecurityGroupIDs são IDs de Security Groups existentes
	// Se especificado, não cria novos Security Groups
	// +optional
	ExistingSecurityGroupIDs []string `json:"existingSecurityGroupIDs,omitempty"`

	// ===========================================================================
	// Campos para CRIAR novos recursos (usados apenas se não passar IDs existentes)
	// ===========================================================================

	// VpcCIDR é o bloco CIDR da VPC (ex: 10.0.0.0/16)
	// Obrigatório apenas se não especificar existingVpcID
	// +optional
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$`
	VpcCIDR string `json:"vpcCIDR,omitempty"`

	// VpcName é o nome da VPC (opcional, usa metadata.name se não especificado)
	// +optional
	VpcName string `json:"vpcName,omitempty"`

	// AvailabilityZones lista as AZs onde criar subnets
	// Obrigatório apenas se não especificar existingSubnetIDs
	// +optional
	// +kubebuilder:validation:MaxItems=6
	AvailabilityZones []string `json:"availabilityZones,omitempty"`

	// PublicSubnets define as subnets públicas (com acesso à internet via IGW)
	// Obrigatório apenas se não especificar existingSubnetIDs
	// +optional
	PublicSubnets []SubnetConfig `json:"publicSubnets,omitempty"`

	// PrivateSubnets define as subnets privadas (com acesso à internet via NAT)
	// +optional
	PrivateSubnets []SubnetConfig `json:"privateSubnets,omitempty"`

	// NATGateway configura o NAT Gateway para subnets privadas
	// +optional
	NATGateway *NATGatewayConfig `json:"natGateway,omitempty"`

	// DefaultSecurityGroups define security groups padrão a serem criados
	// +optional
	DefaultSecurityGroups []SecurityGroupConfig `json:"defaultSecurityGroups,omitempty"`

	// EnableDNSHostnames habilita DNS hostnames na VPC
	// +optional
	// +kubebuilder:default=true
	EnableDNSHostnames bool `json:"enableDNSHostnames,omitempty"`

	// EnableDNSSupport habilita suporte a DNS na VPC
	// +optional
	// +kubebuilder:default=true
	EnableDNSSupport bool `json:"enableDNSSupport,omitempty"`

	// Tags são tags adicionais aplicadas a todos os recursos
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy define o comportamento ao deletar o CR
	// +optional
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// ===========================================================================
	// Campos para EC2 Instance (Bastion/Jump Host)
	// ===========================================================================

	// BastionInstance configura uma instância EC2 bastion para acesso SSH
	// +optional
	BastionInstance *BastionInstanceConfig `json:"bastionInstance,omitempty"`
}

// BastionInstanceConfig define a configuração da instância bastion EC2
type BastionInstanceConfig struct {
	// Enabled habilita a criação de uma instância bastion
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Name é o nome da instância (opcional, usa metadata.name-bastion se não especificado)
	// +optional
	Name string `json:"name,omitempty"`

	// InstanceType é o tipo da instância EC2 (ex: t3.micro, t3.small)
	// +kubebuilder:default="t3.micro"
	// +optional
	InstanceType string `json:"instanceType,omitempty"`

	// ImageID é o ID da AMI para usar (opcional, usa Amazon Linux 2 se não especificado)
	// +optional
	ImageID string `json:"imageID,omitempty"`

	// KeyName é o nome do Key Pair para acesso SSH (obrigatório se enabled=true)
	// +optional
	KeyName string `json:"keyName,omitempty"`

	// SSHAllowedCIDRs são os blocos CIDR permitidos para SSH (porta 22)
	// +optional
	// +kubebuilder:default={"0.0.0.0/0"}
	SSHAllowedCIDRs []string `json:"sshAllowedCIDRs,omitempty"`

	// AssociatePublicIP associa um IP público à instância
	// +kubebuilder:default=true
	// +optional
	AssociatePublicIP bool `json:"associatePublicIP,omitempty"`

	// RootVolumeSize é o tamanho do volume root em GB
	// +kubebuilder:default=20
	// +kubebuilder:validation:Minimum=8
	// +kubebuilder:validation:Maximum=16384
	// +optional
	RootVolumeSize int32 `json:"rootVolumeSize,omitempty"`

	// UserData é o script cloud-init para executar no boot da instância
	// Suporta shell scripts (#!) ou cloud-config YAML (#cloud-config)
	// O conteúdo será automaticamente codificado em base64
	// +optional
	UserData string `json:"userData,omitempty"`

	// UserDataSecretRef referencia um Secret contendo o userData
	// Alternativa ao campo userData para scripts maiores ou sensíveis
	// O Secret deve ter uma chave "userData" com o conteúdo do script
	// +optional
	UserDataSecretRef *SecretKeyReference `json:"userDataSecretRef,omitempty"`

	// Tags são tags específicas da instância bastion
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// SecretKeyReference referencia uma chave específica em um Secret
type SecretKeyReference struct {
	// Name é o nome do Secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace é o namespace do Secret (opcional, usa o namespace do CR se não especificado)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Key é a chave dentro do Secret (default: "userData")
	// +optional
	// +kubebuilder:default="userData"
	Key string `json:"key,omitempty"`
}

// SubnetConfig define a configuração de uma subnet
type SubnetConfig struct {
	// CIDR é o bloco CIDR da subnet (ex: 10.0.1.0/24)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$`
	CIDR string `json:"cidr"`

	// AvailabilityZone é a AZ onde criar a subnet
	// +kubebuilder:validation:Required
	AvailabilityZone string `json:"availabilityZone"`

	// Name é o nome da subnet (opcional)
	// +optional
	Name string `json:"name,omitempty"`

	// Tags são tags específicas desta subnet
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// NATGatewayConfig define a configuração do NAT Gateway
type NATGatewayConfig struct {
	// Enabled habilita a criação de NAT Gateway
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// HighAvailability cria um NAT Gateway por AZ (mais caro, mais resiliente)
	// Se false, cria apenas um NAT Gateway compartilhado
	// +optional
	// +kubebuilder:default=false
	HighAvailability bool `json:"highAvailability,omitempty"`
}

// SecurityGroupConfig define a configuração de um Security Group
type SecurityGroupConfig struct {
	// Name é o nome do Security Group
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description é a descrição do Security Group
	// +optional
	Description string `json:"description,omitempty"`

	// IngressRules são as regras de entrada
	// +optional
	IngressRules []SecurityGroupRuleConfig `json:"ingressRules,omitempty"`

	// EgressRules são as regras de saída
	// +optional
	EgressRules []SecurityGroupRuleConfig `json:"egressRules,omitempty"`
}

// SecurityGroupRuleConfig define uma regra de Security Group simplificada
type SecurityGroupRuleConfig struct {
	// Protocol é o protocolo (tcp, udp, icmp, -1 para todos)
	// +kubebuilder:validation:Required
	Protocol string `json:"protocol"`

	// Port é a porta (ou porta inicial se toPort especificado)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// ToPort é a porta final (opcional, usa Port se não especificado)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	ToPort int32 `json:"toPort,omitempty"`

	// CIDR é o bloco CIDR permitido (ex: 0.0.0.0/0)
	// +kubebuilder:validation:Required
	CIDR string `json:"cidr"`

	// Description é a descrição da regra
	// +optional
	Description string `json:"description,omitempty"`
}

// ComputeStackStatus define o estado observado do ComputeStack
type ComputeStackStatus struct {
	// Ready indica se toda a stack está pronta
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Phase indica a fase atual da stack
	// +optional
	Phase string `json:"phase,omitempty"`

	// VPC contém informações da VPC criada
	// +optional
	VPC *VPCStatusInfo `json:"vpc,omitempty"`

	// InternetGateway contém informações do Internet Gateway
	// +optional
	InternetGateway *IGWStatusInfo `json:"internetGateway,omitempty"`

	// PublicSubnets contém informações das subnets públicas
	// +optional
	PublicSubnets []SubnetStatusInfo `json:"publicSubnets,omitempty"`

	// PrivateSubnets contém informações das subnets privadas
	// +optional
	PrivateSubnets []SubnetStatusInfo `json:"privateSubnets,omitempty"`

	// NATGateways contém informações dos NAT Gateways
	// +optional
	NATGateways []NATGatewayStatusInfo `json:"natGateways,omitempty"`

	// RouteTables contém informações das Route Tables
	// +optional
	RouteTables []RouteTableStatusInfo `json:"routeTables,omitempty"`

	// SecurityGroups contém informações dos Security Groups
	// +optional
	SecurityGroups []SecurityGroupStatusInfo `json:"securityGroups,omitempty"`

	// BastionSecurityGroup contém informações do Security Group do bastion
	// +optional
	BastionSecurityGroup *SecurityGroupStatusInfo `json:"bastionSecurityGroup,omitempty"`

	// BastionInstance contém informações da instância bastion EC2
	// +optional
	BastionInstance *BastionInstanceStatusInfo `json:"bastionInstance,omitempty"`

	// Conditions são as condições do recurso
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncTime é o timestamp da última sincronização
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Message contém mensagem de status ou erro
	// +optional
	Message string `json:"message,omitempty"`
}

// VPCStatusInfo contém informações de status da VPC
type VPCStatusInfo struct {
	// VPCID é o ID da VPC na AWS
	ID string `json:"id,omitempty"`
	// CIDR é o bloco CIDR da VPC
	CIDR string `json:"cidr,omitempty"`
	// State é o estado da VPC
	State string `json:"state,omitempty"`
}

// IGWStatusInfo contém informações de status do Internet Gateway
type IGWStatusInfo struct {
	// ID é o ID do Internet Gateway na AWS
	ID string `json:"id,omitempty"`
	// State é o estado do IGW
	State string `json:"state,omitempty"`
}

// SubnetStatusInfo contém informações de status de uma subnet
type SubnetStatusInfo struct {
	// ID é o ID da subnet na AWS
	ID string `json:"id,omitempty"`
	// CIDR é o bloco CIDR da subnet
	CIDR string `json:"cidr,omitempty"`
	// AvailabilityZone é a AZ da subnet
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	// Type é o tipo da subnet (public/private)
	Type string `json:"type,omitempty"`
	// State é o estado da subnet
	State string `json:"state,omitempty"`
}

// NATGatewayStatusInfo contém informações de status do NAT Gateway
type NATGatewayStatusInfo struct {
	// ID é o ID do NAT Gateway na AWS
	ID string `json:"id,omitempty"`
	// ElasticIP é o IP elástico associado
	ElasticIP string `json:"elasticIP,omitempty"`
	// AllocationID é o ID do Elastic IP
	AllocationID string `json:"allocationID,omitempty"`
	// SubnetID é a subnet onde o NAT está
	SubnetID string `json:"subnetID,omitempty"`
	// State é o estado do NAT Gateway
	State string `json:"state,omitempty"`
}

// RouteTableStatusInfo contém informações de status da Route Table
type RouteTableStatusInfo struct {
	// ID é o ID da Route Table na AWS
	ID string `json:"id,omitempty"`
	// Type é o tipo (public/private)
	Type string `json:"type,omitempty"`
	// AssociatedSubnets são as subnets associadas
	AssociatedSubnets []string `json:"associatedSubnets,omitempty"`
}

// SecurityGroupStatusInfo contém informações de status do Security Group
type SecurityGroupStatusInfo struct {
	// ID é o ID do Security Group na AWS
	ID string `json:"id,omitempty"`
	// Name é o nome do Security Group
	Name string `json:"name,omitempty"`
}

// BastionInstanceStatusInfo contém informações de status da instância bastion EC2
type BastionInstanceStatusInfo struct {
	// ID é o ID da instância EC2 na AWS
	ID string `json:"id,omitempty"`
	// Name é o nome da instância
	Name string `json:"name,omitempty"`
	// PublicIP é o IP público da instância
	PublicIP string `json:"publicIP,omitempty"`
	// PrivateIP é o IP privado da instância
	PrivateIP string `json:"privateIP,omitempty"`
	// State é o estado da instância (pending, running, stopped, terminated)
	State string `json:"state,omitempty"`
	// InstanceType é o tipo da instância
	InstanceType string `json:"instanceType,omitempty"`
	// AvailabilityZone é a AZ onde a instância está rodando
	AvailabilityZone string `json:"availabilityZone,omitempty"`
	// SSHCommand é o comando SSH para conectar na instância
	SSHCommand string `json:"sshCommand,omitempty"`
	// KeyPairName é o nome do key pair usado (pode ser gerado automaticamente)
	KeyPairName string `json:"keyPairName,omitempty"`
	// KeyPairGenerated indica se o key pair foi gerado automaticamente pelo operator
	KeyPairGenerated bool `json:"keyPairGenerated,omitempty"`
	// SSHKeySecretName é o nome do Secret que contém a chave privada SSH (se gerada)
	SSHKeySecretName string `json:"sshKeySecretName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cs;cstack
// +kubebuilder:printcolumn:name="VPC",type="string",JSONPath=".status.vpc.id",description="VPC ID"
// +kubebuilder:printcolumn:name="CIDR",type="string",JSONPath=".spec.vpcCIDR",description="VPC CIDR"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Current phase"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Stack ready"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ComputeStack é o Schema para a API computestacks
// Cria toda a infraestrutura de rede AWS em um único recurso:
// VPC, Subnets, Internet Gateway, NAT Gateway, Route Tables e Security Groups
type ComputeStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComputeStackSpec   `json:"spec,omitempty"`
	Status ComputeStackStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ComputeStackList contém uma lista de ComputeStack
type ComputeStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComputeStack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ComputeStack{}, &ComputeStackList{})
}
