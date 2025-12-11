package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DynamoDBTableSpec defines the desired state of DynamoDBTable
type DynamoDBTableSpec struct {
	// TableName is the name of the DynamoDB table
	TableName string `json:"tableName"`

	// ProviderRef references the AWSProvider
	ProviderRef ProviderReference `json:"providerRef"`

	// BillingMode controls how you are charged for read and write throughput
	// +kubebuilder:validation:Enum=PROVISIONED;PAY_PER_REQUEST
	// +optional
	BillingMode string `json:"billingMode,omitempty"`

	// HashKey is the partition key attribute
	HashKey AttributeDefinition `json:"hashKey"`

	// RangeKey is the sort key attribute (optional)
	// +optional
	RangeKey *AttributeDefinition `json:"rangeKey,omitempty"`

	// Attributes defines all attributes used in keys
	Attributes []AttributeDefinition `json:"attributes,omitempty"`

	// GlobalSecondaryIndexes defines GSIs
	// +optional
	GlobalSecondaryIndexes []GlobalSecondaryIndex `json:"globalSecondaryIndexes,omitempty"`

	// StreamEnabled enables DynamoDB Streams
	// +optional
	StreamEnabled bool `json:"streamEnabled,omitempty"`

	// StreamViewType determines what information is written to the stream
	// +kubebuilder:validation:Enum=NEW_IMAGE;OLD_IMAGE;NEW_AND_OLD_IMAGES;KEYS_ONLY
	// +optional
	StreamViewType string `json:"streamViewType,omitempty"`

	// PointInTimeRecovery enables point-in-time recovery
	// +optional
	PointInTimeRecovery bool `json:"pointInTimeRecovery,omitempty"`

	// Tags for the table
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines behavior on CR deletion
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// AttributeDefinition defines a DynamoDB attribute
type AttributeDefinition struct {
	// Name of the attribute
	Name string `json:"name"`

	// Type of the attribute (S=String, N=Number, B=Binary)
	// +kubebuilder:validation:Enum=S;N;B
	Type string `json:"type"`
}

// GlobalSecondaryIndex defines a GSI
type GlobalSecondaryIndex struct {
	// IndexName is the name of the index
	IndexName string `json:"indexName"`

	// HashKey for the index
	HashKey string `json:"hashKey"`

	// RangeKey for the index (optional)
	// +optional
	RangeKey string `json:"rangeKey,omitempty"`

	// ProjectionType determines which attributes are projected
	// +kubebuilder:validation:Enum=ALL;KEYS_ONLY;INCLUDE
	ProjectionType string `json:"projectionType"`

	// NonKeyAttributes to include in projection (for INCLUDE type)
	// +optional
	NonKeyAttributes []string `json:"nonKeyAttributes,omitempty"`
}

// DynamoDBTableStatus defines the observed state of DynamoDBTable
type DynamoDBTableStatus struct {
	// Ready indicates if the table is ready
	Ready bool `json:"ready"`

	// TableARN is the ARN of the table
	// +optional
	TableARN string `json:"tableARN,omitempty"`

	// TableStatus is the current status
	// +optional
	TableStatus string `json:"tableStatus,omitempty"`

	// ItemCount is the approximate number of items
	// +optional
	ItemCount int64 `json:"itemCount,omitempty"`

	// TableSizeBytes is the approximate size
	// +optional
	TableSizeBytes int64 `json:"tableSizeBytes,omitempty"`

	// StreamARN is the ARN of the stream (if enabled)
	// +optional
	StreamARN string `json:"streamARN,omitempty"`

	// LastSyncTime is the last time the table was synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ddb;ddbtable
// +kubebuilder:printcolumn:name="Table",type=string,JSONPath=`.spec.tableName`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.tableStatus`
// +kubebuilder:printcolumn:name="Items",type=integer,JSONPath=`.status.itemCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DynamoDBTable is the Schema for the dynamodbtables API
type DynamoDBTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynamoDBTableSpec   `json:"spec,omitempty"`
	Status DynamoDBTableStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DynamoDBTableList contains a list of DynamoDBTable
type DynamoDBTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynamoDBTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DynamoDBTable{}, &DynamoDBTableList{})
}
