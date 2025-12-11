package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/sqs"
)

// CRToDomainQueue converts a SQSQueue CR to domain model
func CRToDomainQueue(cr *infrav1alpha1.SQSQueue) *sqs.Queue {
	queue := &sqs.Queue{
		Name:                          cr.Spec.QueueName,
		FifoQueue:                     cr.Spec.FifoQueue,
		ContentBasedDeduplication:     cr.Spec.ContentBasedDeduplication,
		DelaySeconds:                  cr.Spec.DelaySeconds,
		MaximumMessageSize:            cr.Spec.MaximumMessageSize,
		MessageRetentionPeriod:        cr.Spec.MessageRetentionPeriod,
		VisibilityTimeout:             cr.Spec.VisibilityTimeout,
		ReceiveMessageWaitTimeSeconds: cr.Spec.ReceiveMessageWaitTimeSeconds,
		KMSMasterKeyID:                cr.Spec.KMSMasterKeyID,
		KMSDataKeyReusePeriodSeconds:  cr.Spec.KMSDataKeyReusePeriodSeconds,
		Tags:                          cr.Spec.Tags,
		DeletionPolicy:                cr.Spec.DeletionPolicy,
	}

	// Map Dead Letter Queue configuration
	if cr.Spec.DeadLetterQueue != nil {
		queue.DeadLetterQueue = &sqs.DeadLetterQueueConfig{
			TargetArn:       cr.Spec.DeadLetterQueue.TargetArn,
			MaxReceiveCount: cr.Spec.DeadLetterQueue.MaxReceiveCount,
		}
	}

	// Copy status fields if available
	if cr.Status.QueueURL != "" {
		queue.URL = cr.Status.QueueURL
	}
	if cr.Status.QueueARN != "" {
		queue.ARN = cr.Status.QueueARN
	}

	queue.ApproximateNumberOfMessages = cr.Status.ApproximateNumberOfMessages
	queue.ApproximateNumberOfMessagesNotVisible = cr.Status.ApproximateNumberOfMessagesNotVisible

	if cr.Status.LastSyncTime != nil {
		lastSyncTime := cr.Status.LastSyncTime.Time
		queue.LastSyncTime = &lastSyncTime
	}

	return queue
}

// DomainQueueToStatus converts domain model to CR status
func DomainQueueToStatus(queue *sqs.Queue) infrav1alpha1.SQSQueueStatus {
	status := infrav1alpha1.SQSQueueStatus{
		Ready:                                 queue.IsReady(),
		QueueURL:                              queue.URL,
		QueueARN:                              queue.ARN,
		ApproximateNumberOfMessages:           queue.ApproximateNumberOfMessages,
		ApproximateNumberOfMessagesNotVisible: queue.ApproximateNumberOfMessagesNotVisible,
	}

	// Set last sync time
	now := metav1.NewTime(time.Now())
	status.LastSyncTime = &now

	return status
}
