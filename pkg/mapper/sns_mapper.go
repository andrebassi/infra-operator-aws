package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/sns"
)

// CRToDomainTopic converts a SNSTopic CR to domain model
func CRToDomainTopic(cr *infrav1alpha1.SNSTopic) *sns.Topic {
	topic := &sns.Topic{
		Name:                      cr.Spec.TopicName,
		DisplayName:               cr.Spec.DisplayName,
		FifoTopic:                 cr.Spec.FifoTopic,
		ContentBasedDeduplication: cr.Spec.ContentBasedDeduplication,
		KMSMasterKeyID:            cr.Spec.KmsMasterKeyId,
		DeliveryPolicy:            cr.Spec.DeliveryPolicy,
		Tags:                      cr.Spec.Tags,
		DeletionPolicy:            cr.Spec.DeletionPolicy,
	}

	// Map subscriptions
	for _, sub := range cr.Spec.Subscriptions {
		topic.Subscriptions = append(topic.Subscriptions, sns.Subscription{
			Protocol:           sub.Protocol,
			Endpoint:           sub.Endpoint,
			FilterPolicy:       sub.FilterPolicy,
			RawMessageDelivery: sub.RawMessageDelivery,
			DeadLetterQueueArn: sub.DeadLetterQueueArn,
		})
	}

	// Copy status fields if available
	if cr.Status.TopicArn != "" {
		topic.ARN = cr.Status.TopicArn
	}

	topic.SubscriptionsConfirmed = cr.Status.SubscriptionsConfirmed
	topic.SubscriptionsPending = cr.Status.SubscriptionsPending

	if cr.Status.LastSyncTime != nil {
		lastSyncTime := cr.Status.LastSyncTime.Time
		topic.LastSyncTime = &lastSyncTime
	}

	return topic
}

// DomainTopicToStatus converts domain model to CR status
func DomainTopicToStatus(topic *sns.Topic) infrav1alpha1.SNSTopicStatus {
	status := infrav1alpha1.SNSTopicStatus{
		Ready:                  topic.IsReady(),
		TopicArn:               topic.ARN,
		SubscriptionsConfirmed: topic.SubscriptionsConfirmed,
		SubscriptionsPending:   topic.SubscriptionsPending,
	}

	// Set subscription ARNs
	for _, sub := range topic.Subscriptions {
		if sub.ARN != "" {
			status.SubscriptionArns = append(status.SubscriptionArns, sub.ARN)
		}
	}

	// Set last sync time
	now := metav1.NewTime(time.Now())
	status.LastSyncTime = &now

	return status
}
