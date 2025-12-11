package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SQSQueueSpec defines the desired state of SQSQueue
type SQSQueueSpec struct {
	// QueueName is the name of the SQS queue
	QueueName string `json:"queueName"`

	// ProviderRef references the AWSProvider
	ProviderRef ProviderReference `json:"providerRef"`

	// FifoQueue designates a queue as FIFO
	// +optional
	FifoQueue bool `json:"fifoQueue,omitempty"`

	// ContentBasedDeduplication enables content-based deduplication (FIFO only)
	// +optional
	ContentBasedDeduplication bool `json:"contentBasedDeduplication,omitempty"`

	// DelaySeconds - delivery delay in seconds (0-900)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=900
	DelaySeconds int32 `json:"delaySeconds,omitempty"`

	// MaximumMessageSize in bytes (1024-262144)
	// +optional
	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=262144
	MaximumMessageSize int32 `json:"maximumMessageSize,omitempty"`

	// MessageRetentionPeriod in seconds (60-1209600)
	// +optional
	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:validation:Maximum=1209600
	MessageRetentionPeriod int32 `json:"messageRetentionPeriod,omitempty"`

	// VisibilityTimeout in seconds (0-43200)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=43200
	VisibilityTimeout int32 `json:"visibilityTimeout,omitempty"`

	// ReceiveMessageWaitTimeSeconds - long polling (0-20)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=20
	ReceiveMessageWaitTimeSeconds int32 `json:"receiveMessageWaitTimeSeconds,omitempty"`

	// DeadLetterQueue configuration
	// +optional
	DeadLetterQueue *DeadLetterQueueConfig `json:"deadLetterQueue,omitempty"`

	// KMSMasterKeyID for encryption
	// +optional
	KMSMasterKeyID string `json:"kmsMasterKeyId,omitempty"`

	// KMSDataKeyReusePeriodSeconds (60-86400)
	// +optional
	KMSDataKeyReusePeriodSeconds int32 `json:"kmsDataKeyReusePeriodSeconds,omitempty"`

	// Tags for the queue
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy determines behavior on CR deletion
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// DeadLetterQueueConfig defines DLQ settings
type DeadLetterQueueConfig struct {
	// TargetArn is the ARN of the dead-letter queue
	TargetArn string `json:"targetArn"`

	// MaxReceiveCount is the number of receives before sending to DLQ
	MaxReceiveCount int32 `json:"maxReceiveCount"`
}

// SQSQueueStatus defines the observed state of SQSQueue
type SQSQueueStatus struct {
	// Ready indicates if the queue is ready
	Ready bool `json:"ready"`

	// QueueURL is the URL of the queue
	// +optional
	QueueURL string `json:"queueURL,omitempty"`

	// QueueARN is the ARN of the queue
	// +optional
	QueueARN string `json:"queueARN,omitempty"`

	// ApproximateNumberOfMessages
	// +optional
	ApproximateNumberOfMessages int64 `json:"approximateNumberOfMessages,omitempty"`

	// ApproximateNumberOfMessagesNotVisible
	// +optional
	ApproximateNumberOfMessagesNotVisible int64 `json:"approximateNumberOfMessagesNotVisible,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=sqs
// +kubebuilder:printcolumn:name="Queue",type=string,JSONPath=`.spec.queueName`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Messages",type=integer,JSONPath=`.status.approximateNumberOfMessages`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SQSQueue is the Schema for the sqsqueues API
type SQSQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SQSQueueSpec   `json:"spec,omitempty"`
	Status SQSQueueStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SQSQueueList contains a list of SQSQueue
type SQSQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQSQueue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SQSQueue{}, &SQSQueueList{})
}
