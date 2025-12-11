package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RDSInstanceSpec defines the desired state of RDSInstance
type RDSInstanceSpec struct {
	ProviderRef ProviderReference `json:"providerRef"`

	// DBInstanceIdentifier is the DB instance identifier
	DBInstanceIdentifier string `json:"dbInstanceIdentifier"`

	// Engine is the database engine (mysql, postgres, mariadb, etc)
	Engine string `json:"engine"`

	// EngineVersion is the version of the database engine
	EngineVersion string `json:"engineVersion,omitempty"`

	// DBInstanceClass is the compute and memory capacity (db.t3.micro, etc)
	DBInstanceClass string `json:"dbInstanceClass"`

	// AllocatedStorage in GB
	AllocatedStorage int32 `json:"allocatedStorage"`

	// MasterUsername for the database
	MasterUsername string `json:"masterUsername"`

	// MasterUserPasswordSecretRef references a secret containing the password
	MasterUserPasswordSecretRef *SecretReference `json:"masterUserPasswordSecretRef,omitempty"`

	// MasterUserPassword - direct password (not recommended for production)
	MasterUserPassword string `json:"masterUserPassword,omitempty"`

	// DBName is the name of the initial database
	DBName string `json:"dbName,omitempty"`

	// Port for the database
	Port int32 `json:"port,omitempty"`

	// MultiAZ specifies if this is a Multi-AZ deployment
	MultiAZ bool `json:"multiAZ,omitempty"`

	// PubliclyAccessible specifies if the DB is publicly accessible
	PubliclyAccessible bool `json:"publiclyAccessible,omitempty"`

	// StorageEncrypted specifies if storage is encrypted
	StorageEncrypted bool `json:"storageEncrypted,omitempty"`

	// BackupRetentionPeriod in days (0-35)
	BackupRetentionPeriod int32 `json:"backupRetentionPeriod,omitempty"`

	// PreferredBackupWindow in format hh24:mi-hh24:mi
	PreferredBackupWindow string `json:"preferredBackupWindow,omitempty"`

	// Tags for the RDS instance
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines what happens when CR is deleted
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// SkipFinalSnapshot if true, skips final snapshot on deletion
	SkipFinalSnapshot bool `json:"skipFinalSnapshot,omitempty"`
}

type SecretReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// RDSInstanceStatus defines the observed state of RDSInstance
type RDSInstanceStatus struct {
	Ready bool `json:"ready"`

	// DBInstanceArn is the ARN of the RDS instance
	DBInstanceArn string `json:"dbInstanceArn,omitempty"`

	// Endpoint is the connection endpoint
	Endpoint string `json:"endpoint,omitempty"`

	// Port is the connection port
	Port int32 `json:"port,omitempty"`

	// Status is the current status (available, creating, modifying, etc)
	Status string `json:"status,omitempty"`

	// EngineVersion is the actual engine version running
	EngineVersion string `json:"engineVersion,omitempty"`

	// AllocatedStorage in GB
	AllocatedStorage int32 `json:"allocatedStorage,omitempty"`

	// LastSyncTime is when the instance was last synced
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=rds
// +kubebuilder:printcolumn:name="Instance",type=string,JSONPath=`.spec.dbInstanceIdentifier`
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.engine`
// +kubebuilder:printcolumn:name="Class",type=string,JSONPath=`.spec.dbInstanceClass`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RDSInstance is the Schema for the rdsinstances API
type RDSInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RDSInstanceSpec   `json:"spec,omitempty"`
	Status RDSInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RDSInstanceList contains a list of RDSInstance
type RDSInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RDSInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RDSInstance{}, &RDSInstanceList{})
}
