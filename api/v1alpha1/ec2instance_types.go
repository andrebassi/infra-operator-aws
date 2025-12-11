package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EC2InstanceSpec defines the desired state of EC2Instance
type EC2InstanceSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// InstanceName is a friendly name for the instance
	// +kubebuilder:validation:Required
	InstanceName string `json:"instanceName"`

	// InstanceType (t3.micro, t3.small, m5.large, etc)
	// +kubebuilder:validation:Required
	InstanceType string `json:"instanceType"`

	// AMI ID
	// +kubebuilder:validation:Required
	ImageID string `json:"imageID"`

	// KeyName for SSH access
	// +optional
	KeyName string `json:"keyName,omitempty"`

	// SubnetID
	// +optional
	SubnetID string `json:"subnetID,omitempty"`

	// SecurityGroupIDs
	// +optional
	SecurityGroupIDs []string `json:"securityGroupIDs,omitempty"`

	// IAMInstanceProfile
	// +optional
	IAMInstanceProfile string `json:"iamInstanceProfile,omitempty"`

	// UserData script (base64 encoded will be handled by controller)
	// +optional
	UserData string `json:"userData,omitempty"`

	// BlockDeviceMappings for EBS volumes
	// +optional
	BlockDeviceMappings []BlockDeviceMapping `json:"blockDeviceMappings,omitempty"`

	// Monitoring enables detailed CloudWatch monitoring
	// +optional
	Monitoring bool `json:"monitoring,omitempty"`

	// DisableAPITermination
	// +optional
	DisableAPITermination bool `json:"disableApiTermination,omitempty"`

	// EBSOptimized
	// +optional
	EBSOptimized bool `json:"ebsOptimized,omitempty"`

	// Tags
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain;Stop
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// EnableConsoleOutput habilita a coleta de logs do console da EC2
	// Os logs serão armazenados no status.consoleOutput
	// +optional
	EnableConsoleOutput bool `json:"enableConsoleOutput,omitempty"`
}

// BlockDeviceMapping defines an EBS volume mapping
type BlockDeviceMapping struct {
	// DeviceName (e.g., /dev/sda1, /dev/xvdf)
	DeviceName string `json:"deviceName"`

	// EBS configuration
	// +optional
	EBS *EBSBlockDevice `json:"ebs,omitempty"`
}

// EBSBlockDevice defines EBS volume configuration
type EBSBlockDevice struct {
	// VolumeSize in GB
	VolumeSize int32 `json:"volumeSize"`

	// VolumeType (gp2, gp3, io1, io2, st1, sc1)
	// +optional
	// +kubebuilder:validation:Enum=gp2;gp3;io1;io2;st1;sc1;standard
	// +kubebuilder:default=gp3
	VolumeType string `json:"volumeType,omitempty"`

	// IOPS for io1/io2
	// +optional
	IOPS int32 `json:"iops,omitempty"`

	// DeleteOnTermination
	// +optional
	// +kubebuilder:default=true
	DeleteOnTermination bool `json:"deleteOnTermination,omitempty"`

	// Encrypted
	// +optional
	Encrypted bool `json:"encrypted,omitempty"`

	// KMSKeyID for encryption
	// +optional
	KMSKeyID string `json:"kmsKeyID,omitempty"`
}

// EC2InstanceStatus defines the observed state of EC2Instance
type EC2InstanceStatus struct {
	// Conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// InstanceID
	// +optional
	InstanceID string `json:"instanceID,omitempty"`

	// InstanceState (pending, running, stopping, stopped, terminated)
	// +optional
	InstanceState string `json:"instanceState,omitempty"`

	// PrivateIP
	// +optional
	PrivateIP string `json:"privateIP,omitempty"`

	// PublicIP
	// +optional
	PublicIP string `json:"publicIP,omitempty"`

	// PrivateDNS
	// +optional
	PrivateDNS string `json:"privateDNS,omitempty"`

	// PublicDNS
	// +optional
	PublicDNS string `json:"publicDNS,omitempty"`

	// AvailabilityZone
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// LaunchTime
	// +optional
	LaunchTime *metav1.Time `json:"launchTime,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// ConsoleOutput contém as últimas linhas do console output da EC2
	// Preenchido apenas quando spec.enableConsoleOutput=true
	// +optional
	ConsoleOutput string `json:"consoleOutput,omitempty"`

	// ConsoleOutputTimestamp é o timestamp do último console output
	// +optional
	ConsoleOutputTimestamp *metav1.Time `json:"consoleOutputTimestamp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ec2
// +kubebuilder:printcolumn:name="Instance",type=string,JSONPath=`.spec.instanceName`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.instanceType`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.instanceState`
// +kubebuilder:printcolumn:name="InstanceID",type=string,JSONPath=`.status.instanceID`
// +kubebuilder:printcolumn:name="PrivateIP",type=string,JSONPath=`.status.privateIP`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// EC2Instance is the Schema for the ec2instances API
type EC2Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EC2InstanceSpec   `json:"spec,omitempty"`
	Status EC2InstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EC2InstanceList contains a list of EC2Instance
type EC2InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EC2Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EC2Instance{}, &EC2InstanceList{})
}
