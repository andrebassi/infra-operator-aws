package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SNSTopicSpec defines the desired state of SNSTopic
type SNSTopicSpec struct {
	// ProviderRef references the AWSProvider to use
	// +kubebuilder:validation:Required
	ProviderRef ProviderReference `json:"providerRef"`

	// TopicName
	// +kubebuilder:validation:Required
	TopicName string `json:"topicName"`

	// DisplayName (for SMS messages)
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// FifoTopic
	// +optional
	FifoTopic bool `json:"fifoTopic,omitempty"`

	// ContentBasedDeduplication (only for FIFO topics)
	// +optional
	ContentBasedDeduplication bool `json:"contentBasedDeduplication,omitempty"`

	// KmsMasterKeyId for encryption
	// +optional
	KmsMasterKeyId string `json:"kmsMasterKeyId,omitempty"`

	// DeliveryPolicy
	// +optional
	DeliveryPolicy string `json:"deliveryPolicy,omitempty"`

	// Subscriptions to create
	// +optional
	Subscriptions []SNSSubscription `json:"subscriptions,omitempty"`

	// Tags
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// DeletionPolicy
	// +optional
	// +kubebuilder:validation:Enum=Delete;Retain
	// +kubebuilder:default=Delete
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

// SNSSubscription defines a subscription to the topic
type SNSSubscription struct {
	// Protocol (http, https, email, email-json, sms, sqs, lambda, application, firehose)
	// +kubebuilder:validation:Enum=http;https;email;email-json;sms;sqs;lambda;application;firehose
	Protocol string `json:"protocol"`

	// Endpoint (URL, email, phone number, SQS ARN, Lambda ARN, etc)
	Endpoint string `json:"endpoint"`

	// FilterPolicy in JSON format
	// +optional
	FilterPolicy string `json:"filterPolicy,omitempty"`

	// RawMessageDelivery
	// +optional
	RawMessageDelivery bool `json:"rawMessageDelivery,omitempty"`

	// DeadLetterQueueArn
	// +optional
	DeadLetterQueueArn string `json:"deadLetterQueueArn,omitempty"`
}

// SNSTopicStatus defines the observed state of SNSTopic
type SNSTopicStatus struct {
	// Conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// TopicArn
	// +optional
	TopicArn string `json:"topicArn,omitempty"`

	// SubscriptionArns created
	// +optional
	SubscriptionArns []string `json:"subscriptionArns,omitempty"`

	// SubscriptionsConfirmed count
	// +optional
	SubscriptionsConfirmed int32 `json:"subscriptionsConfirmed,omitempty"`

	// SubscriptionsPending count
	// +optional
	SubscriptionsPending int32 `json:"subscriptionsPending,omitempty"`

	// LastSyncTime
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=sns
// +kubebuilder:printcolumn:name="Topic",type=string,JSONPath=`.spec.topicName`
// +kubebuilder:printcolumn:name="FIFO",type=boolean,JSONPath=`.spec.fifoTopic`
// +kubebuilder:printcolumn:name="Subscriptions",type=integer,JSONPath=`.status.subscriptionsConfirmed`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SNSTopic is the Schema for the snstopics API
type SNSTopic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SNSTopicSpec   `json:"spec,omitempty"`
	Status SNSTopicStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SNSTopicList contains a list of SNSTopic
type SNSTopicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SNSTopic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SNSTopic{}, &SNSTopicList{})
}
