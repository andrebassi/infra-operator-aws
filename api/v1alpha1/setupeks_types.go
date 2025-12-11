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

// SetupEKSSpec define o estado desejado do SetupEKS
// Um SetupEKS cria toda a infraestrutura necessária para um cluster EKS:
// VPC, Subnets (públicas e privadas), Internet Gateway, NAT Gateway,
// Route Tables, Security Groups, IAM Roles, EKS Cluster e Node Groups
type SetupEKSSpec struct {
	// ProviderRef referencia o AWSProvider para autenticação
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// ===========================================================================
	// Configuração do Cluster EKS
	// ===========================================================================

	// ClusterName é o nome do cluster EKS (usa metadata.name se não especificado)
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// KubernetesVersion é a versão do Kubernetes (ex: 1.28, 1.29, 1.30)
	// +kubebuilder:default="1.29"
	// +kubebuilder:validation:Pattern=`^[0-9]+\.[0-9]+$`
	// +optional
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`

	// EndpointAccess define o acesso ao endpoint da API do cluster
	// +optional
	EndpointAccess *EndpointAccessConfig `json:"endpointAccess,omitempty"`

	// ClusterLogging habilita logs do cluster no CloudWatch
	// +optional
	ClusterLogging *ClusterLoggingConfig `json:"clusterLogging,omitempty"`

	// EncryptionConfig configuração de criptografia com KMS
	// +optional
	EncryptionConfig *EKSEncryptionConfig `json:"encryptionConfig,omitempty"`

	// ===========================================================================
	// Configuração de Rede (VPC)
	// ===========================================================================

	// VpcCIDR é o bloco CIDR da VPC (ex: 10.0.0.0/16)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$`
	VpcCIDR string `json:"vpcCIDR"`

	// AvailabilityZones lista as AZs onde criar subnets (mínimo 2 para EKS)
	// +kubebuilder:validation:MinItems=2
	// +kubebuilder:validation:MaxItems=6
	// +optional
	AvailabilityZones []string `json:"availabilityZones,omitempty"`

	// PublicSubnetCIDRs são os CIDRs das subnets públicas (1 por AZ)
	// Se não especificado, calcula automaticamente baseado no VpcCIDR
	// +optional
	PublicSubnetCIDRs []string `json:"publicSubnetCIDRs,omitempty"`

	// PrivateSubnetCIDRs são os CIDRs das subnets privadas (1 por AZ)
	// Se não especificado, calcula automaticamente baseado no VpcCIDR
	// +optional
	PrivateSubnetCIDRs []string `json:"privateSubnetCIDRs,omitempty"`

	// NATGatewayMode define como criar NAT Gateways
	// +kubebuilder:default="Single"
	// +kubebuilder:validation:Enum=Single;HighAvailability;None
	// +optional
	NATGatewayMode string `json:"natGatewayMode,omitempty"`

	// ExistingVpcID permite usar uma VPC existente ao invés de criar nova
	// +optional
	ExistingVpcID string `json:"existingVpcID,omitempty"`

	// ExistingSubnetIDs permite usar subnets existentes
	// Deve ter pelo menos 2 subnets em AZs diferentes
	// +optional
	ExistingSubnetIDs []string `json:"existingSubnetIDs,omitempty"`

	// ===========================================================================
	// Configuração de Node Groups (Node Pools)
	// ===========================================================================

	// NodePools define os grupos de nós do cluster
	// +kubebuilder:validation:MinItems=1
	NodePools []NodePoolConfig `json:"nodePools"`

	// DefaultNodePool configuração padrão aplicada a todos os node pools
	// +optional
	DefaultNodePool *NodePoolDefaults `json:"defaultNodePool,omitempty"`

	// ===========================================================================
	// Configuração de Add-ons
	// ===========================================================================

	// Addons lista de add-ons EKS a serem instalados
	// +optional
	Addons []EKSAddonConfig `json:"addons,omitempty"`

	// InstallDefaultAddons instala add-ons essenciais (vpc-cni, coredns, kube-proxy)
	// +kubebuilder:default=true
	// +optional
	InstallDefaultAddons bool `json:"installDefaultAddons,omitempty"`

	// ===========================================================================
	// Configuração de Acesso (RBAC/IAM)
	// ===========================================================================

	// EnableIRSA habilita IAM Roles for Service Accounts
	// +kubebuilder:default=true
	// +optional
	EnableIRSA bool `json:"enableIRSA,omitempty"`

	// AccessEntries configura acesso ao cluster via IAM
	// +optional
	AccessEntries []EKSAccessEntry `json:"accessEntries,omitempty"`

	// ===========================================================================
	// Tags e Políticas
	// ===========================================================================

	// Tags são tags adicionais aplicadas a todos os recursos
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy define o comportamento ao deletar o CR
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Retain
	// +optional
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// EndpointAccessConfig define o acesso ao endpoint da API
type EndpointAccessConfig struct {
	// PublicAccess habilita acesso público ao endpoint
	// +kubebuilder:default=true
	// +optional
	PublicAccess bool `json:"publicAccess,omitempty"`

	// PrivateAccess habilita acesso privado (dentro da VPC)
	// +kubebuilder:default=true
	// +optional
	PrivateAccess bool `json:"privateAccess,omitempty"`

	// PublicAccessCIDRs limita acesso público a CIDRs específicos
	// +optional
	PublicAccessCIDRs []string `json:"publicAccessCIDRs,omitempty"`
}

// ClusterLoggingConfig define os logs do cluster
type ClusterLoggingConfig struct {
	// APIServer habilita logs do API server
	// +optional
	APIServer bool `json:"apiServer,omitempty"`

	// Audit habilita logs de auditoria
	// +optional
	Audit bool `json:"audit,omitempty"`

	// Authenticator habilita logs do authenticator
	// +optional
	Authenticator bool `json:"authenticator,omitempty"`

	// ControllerManager habilita logs do controller manager
	// +optional
	ControllerManager bool `json:"controllerManager,omitempty"`

	// Scheduler habilita logs do scheduler
	// +optional
	Scheduler bool `json:"scheduler,omitempty"`
}

// EKSEncryptionConfig define a criptografia do cluster
type EKSEncryptionConfig struct {
	// Enabled habilita criptografia de secrets
	// +kubebuilder:default=false
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// KMSKeyARN é o ARN da chave KMS (cria uma nova se não especificado)
	// +optional
	KMSKeyARN string `json:"kmsKeyARN,omitempty"`
}

// NodePoolConfig define um grupo de nós
type NodePoolConfig struct {
	// Name é o nome do node pool
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Name string `json:"name"`

	// InstanceTypes são os tipos de instância EC2 (ex: t3.medium, m5.large)
	// +kubebuilder:validation:MinItems=1
	InstanceTypes []string `json:"instanceTypes"`

	// ScalingConfig define a configuração de escala
	// +optional
	ScalingConfig *NodePoolScalingConfig `json:"scalingConfig,omitempty"`

	// CapacityType é o tipo de capacidade (ON_DEMAND ou SPOT)
	// +kubebuilder:default="ON_DEMAND"
	// +kubebuilder:validation:Enum=ON_DEMAND;SPOT
	// +optional
	CapacityType string `json:"capacityType,omitempty"`

	// AMIType é o tipo de AMI (AL2_x86_64, AL2_ARM_64, BOTTLEROCKET_x86_64, etc)
	// +kubebuilder:default="AL2_x86_64"
	// +kubebuilder:validation:Enum=AL2_x86_64;AL2_x86_64_GPU;AL2_ARM_64;BOTTLEROCKET_x86_64;BOTTLEROCKET_ARM_64;CUSTOM
	// +optional
	AMIType string `json:"amiType,omitempty"`

	// DiskSize é o tamanho do disco em GB
	// +kubebuilder:default=50
	// +kubebuilder:validation:Minimum=20
	// +kubebuilder:validation:Maximum=16384
	// +optional
	DiskSize int32 `json:"diskSize,omitempty"`

	// Labels são labels Kubernetes aplicados aos nós
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Taints são taints aplicados aos nós
	// +optional
	Taints []NodeTaintConfig `json:"taints,omitempty"`

	// SubnetSelector define em quais subnets criar os nós
	// +kubebuilder:default="private"
	// +kubebuilder:validation:Enum=private;public;all
	// +optional
	SubnetSelector string `json:"subnetSelector,omitempty"`

	// Tags específicas deste node pool
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// UpdateConfig configuração de atualização dos nós
	// +optional
	UpdateConfig *NodePoolUpdateConfig `json:"updateConfig,omitempty"`
}

// NodePoolDefaults define configurações padrão para todos os node pools
type NodePoolDefaults struct {
	// DiskSize padrão em GB
	// +optional
	DiskSize int32 `json:"diskSize,omitempty"`

	// AMIType padrão
	// +optional
	AMIType string `json:"amiType,omitempty"`

	// CapacityType padrão
	// +optional
	CapacityType string `json:"capacityType,omitempty"`

	// Labels padrão
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Tags padrão
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// NodePoolScalingConfig define a configuração de escala
type NodePoolScalingConfig struct {
	// MinSize é o número mínimo de nós
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +optional
	MinSize int32 `json:"minSize,omitempty"`

	// MaxSize é o número máximo de nós
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	// +optional
	MaxSize int32 `json:"maxSize,omitempty"`

	// DesiredSize é o número desejado de nós
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=0
	// +optional
	DesiredSize int32 `json:"desiredSize,omitempty"`
}

// NodeTaintConfig define um taint
type NodeTaintConfig struct {
	// Key é a chave do taint
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// Value é o valor do taint
	// +optional
	Value string `json:"value,omitempty"`

	// Effect é o efeito do taint
	// +kubebuilder:validation:Enum=NO_SCHEDULE;NO_EXECUTE;PREFER_NO_SCHEDULE
	Effect string `json:"effect"`
}

// NodePoolUpdateConfig define a configuração de atualização
type NodePoolUpdateConfig struct {
	// MaxUnavailable é o número máximo de nós indisponíveis durante update
	// +optional
	MaxUnavailable int32 `json:"maxUnavailable,omitempty"`

	// MaxUnavailablePercentage é a porcentagem máxima de nós indisponíveis
	// +optional
	MaxUnavailablePercentage int32 `json:"maxUnavailablePercentage,omitempty"`
}

// EKSAddonConfig define um add-on EKS
type EKSAddonConfig struct {
	// Name é o nome do add-on (vpc-cni, coredns, kube-proxy, aws-ebs-csi-driver, etc)
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Version é a versão do add-on (usa latest se não especificado)
	// +optional
	Version string `json:"version,omitempty"`

	// ServiceAccountRoleARN é o ARN do IAM role para o service account
	// +optional
	ServiceAccountRoleARN string `json:"serviceAccountRoleARN,omitempty"`

	// ConfigurationValues são valores de configuração em JSON
	// +optional
	ConfigurationValues string `json:"configurationValues,omitempty"`

	// ResolveConflicts define como resolver conflitos
	// +kubebuilder:default="OVERWRITE"
	// +kubebuilder:validation:Enum=OVERWRITE;NONE;PRESERVE
	// +optional
	ResolveConflicts string `json:"resolveConflicts,omitempty"`
}

// EKSAccessEntry define uma entrada de acesso ao cluster
type EKSAccessEntry struct {
	// PrincipalARN é o ARN do IAM principal (user, role, etc)
	// +kubebuilder:validation:Required
	PrincipalARN string `json:"principalARN"`

	// Type é o tipo de acesso
	// +kubebuilder:default="STANDARD"
	// +kubebuilder:validation:Enum=STANDARD;EC2_LINUX;EC2_WINDOWS;FARGATE_LINUX
	// +optional
	Type string `json:"type,omitempty"`

	// KubernetesGroups são grupos Kubernetes para o principal
	// +optional
	KubernetesGroups []string `json:"kubernetesGroups,omitempty"`

	// AccessPolicies são políticas de acesso EKS
	// +optional
	AccessPolicies []string `json:"accessPolicies,omitempty"`
}

// SetupEKSStatus define o estado observado do SetupEKS
type SetupEKSStatus struct {
	// Ready indica se todo o setup está pronto
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Phase indica a fase atual do setup
	// +optional
	Phase string `json:"phase,omitempty"`

	// Message contém mensagem de status ou erro
	// +optional
	Message string `json:"message,omitempty"`

	// ===========================================================================
	// Status da Rede
	// ===========================================================================

	// VPC informações da VPC criada
	// +optional
	VPC *VPCStatusInfo `json:"vpc,omitempty"`

	// InternetGateway informações do Internet Gateway
	// +optional
	InternetGateway *IGWStatusInfo `json:"internetGateway,omitempty"`

	// PublicSubnets informações das subnets públicas
	// +optional
	PublicSubnets []SubnetStatusInfo `json:"publicSubnets,omitempty"`

	// PrivateSubnets informações das subnets privadas
	// +optional
	PrivateSubnets []SubnetStatusInfo `json:"privateSubnets,omitempty"`

	// NATGateways informações dos NAT Gateways
	// +optional
	NATGateways []NATGatewayStatusInfo `json:"natGateways,omitempty"`

	// RouteTables informações das Route Tables
	// +optional
	RouteTables []RouteTableStatusInfo `json:"routeTables,omitempty"`

	// ===========================================================================
	// Status do EKS
	// ===========================================================================

	// Cluster informações do cluster EKS
	// +optional
	Cluster *EKSClusterStatusInfo `json:"cluster,omitempty"`

	// NodePools informações dos node groups
	// +optional
	NodePools []NodePoolStatusInfo `json:"nodePools,omitempty"`

	// Addons informações dos add-ons instalados
	// +optional
	Addons []AddonStatusInfo `json:"addons,omitempty"`

	// ===========================================================================
	// Status do IAM
	// ===========================================================================

	// ClusterRole informações do IAM role do cluster
	// +optional
	ClusterRole *IAMRoleStatusInfo `json:"clusterRole,omitempty"`

	// NodeRole informações do IAM role dos nós
	// +optional
	NodeRole *IAMRoleStatusInfo `json:"nodeRole,omitempty"`

	// ===========================================================================
	// Status dos Security Groups
	// ===========================================================================

	// ClusterSecurityGroup informações do SG do cluster
	// +optional
	ClusterSecurityGroup *SecurityGroupStatusInfo `json:"clusterSecurityGroup,omitempty"`

	// NodeSecurityGroup informações do SG dos nós
	// +optional
	NodeSecurityGroup *SecurityGroupStatusInfo `json:"nodeSecurityGroup,omitempty"`

	// ===========================================================================
	// Conexão
	// ===========================================================================

	// KubeconfigCommand comando para obter kubeconfig
	// +optional
	KubeconfigCommand string `json:"kubeconfigCommand,omitempty"`

	// OIDCIssuerURL é a URL do OIDC provider para IRSA
	// +optional
	OIDCIssuerURL string `json:"oidcIssuerURL,omitempty"`

	// ===========================================================================
	// Metadata
	// ===========================================================================

	// Conditions são as condições do recurso
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncTime é o timestamp da última sincronização
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// EKSClusterStatusInfo contém informações do cluster EKS
type EKSClusterStatusInfo struct {
	// Name é o nome do cluster
	Name string `json:"name,omitempty"`

	// ARN é o ARN do cluster
	ARN string `json:"arn,omitempty"`

	// Endpoint é o endpoint da API do cluster
	Endpoint string `json:"endpoint,omitempty"`

	// CertificateAuthority é o CA do cluster (base64)
	CertificateAuthority string `json:"certificateAuthority,omitempty"`

	// Version é a versão do Kubernetes
	Version string `json:"version,omitempty"`

	// PlatformVersion é a versão da plataforma EKS
	PlatformVersion string `json:"platformVersion,omitempty"`

	// Status é o status do cluster (CREATING, ACTIVE, DELETING, FAILED, UPDATING)
	Status string `json:"status,omitempty"`

	// CreatedAt é quando o cluster foi criado
	CreatedAt string `json:"createdAt,omitempty"`
}

// NodePoolStatusInfo contém informações de um node group
type NodePoolStatusInfo struct {
	// Name é o nome do node group
	Name string `json:"name,omitempty"`

	// ARN é o ARN do node group
	ARN string `json:"arn,omitempty"`

	// Status é o status (CREATING, ACTIVE, UPDATING, DELETING, CREATE_FAILED, DELETE_FAILED)
	Status string `json:"status,omitempty"`

	// CapacityType é ON_DEMAND ou SPOT
	CapacityType string `json:"capacityType,omitempty"`

	// InstanceTypes são os tipos de instância
	InstanceTypes []string `json:"instanceTypes,omitempty"`

	// DesiredSize é o número desejado de nós
	DesiredSize int32 `json:"desiredSize,omitempty"`

	// MinSize é o número mínimo de nós
	MinSize int32 `json:"minSize,omitempty"`

	// MaxSize é o número máximo de nós
	MaxSize int32 `json:"maxSize,omitempty"`

	// Subnets são as subnets onde os nós estão
	Subnets []string `json:"subnets,omitempty"`
}

// AddonStatusInfo contém informações de um add-on
type AddonStatusInfo struct {
	// Name é o nome do add-on
	Name string `json:"name,omitempty"`

	// Version é a versão instalada
	Version string `json:"version,omitempty"`

	// Status é o status (CREATING, ACTIVE, CREATE_FAILED, UPDATING, DELETING, DELETE_FAILED)
	Status string `json:"status,omitempty"`
}

// IAMRoleStatusInfo contém informações de um IAM role
type IAMRoleStatusInfo struct {
	// Name é o nome do role
	Name string `json:"name,omitempty"`

	// ARN é o ARN do role
	ARN string `json:"arn,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=seks;setupeks
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".status.cluster.name",description="EKS Cluster Name"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.kubernetesVersion",description="Kubernetes Version"
// +kubebuilder:printcolumn:name="VPC",type="string",JSONPath=".status.vpc.id",description="VPC ID"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Current phase"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Setup ready"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// SetupEKS é o Schema para a API setupeks
// Cria toda a infraestrutura AWS necessária para um cluster EKS funcional:
// VPC, Subnets, NAT Gateway, IAM Roles, Security Groups, EKS Cluster e Node Groups
type SetupEKS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SetupEKSSpec   `json:"spec,omitempty"`
	Status SetupEKSStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SetupEKSList contém uma lista de SetupEKS
type SetupEKSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SetupEKS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SetupEKS{}, &SetupEKSList{})
}
