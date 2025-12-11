package sqs

import (
	"errors"
	"time"
)

var (
	ErrQueueNotFound    = errors.New("queue not found")
	ErrInvalidQueueName = errors.New("invalid queue name")
)

// Queue represents an SQS queue in the domain
type Queue struct {
	Name                                  string
	URL                                   string
	ARN                                   string
	FifoQueue                             bool
	ContentBasedDeduplication             bool
	DelaySeconds                          int32
	MaximumMessageSize                    int32
	MessageRetentionPeriod                int32
	VisibilityTimeout                     int32
	ReceiveMessageWaitTimeSeconds         int32
	DeadLetterQueue                       *DeadLetterQueueConfig
	KMSMasterKeyID                        string
	KMSDataKeyReusePeriodSeconds          int32
	Tags                                  map[string]string
	ApproximateNumberOfMessages           int64
	ApproximateNumberOfMessagesNotVisible int64
	CreationTime                          *time.Time
	LastSyncTime                          *time.Time
	DeletionPolicy                        string
}

// DeadLetterQueueConfig represents DLQ configuration
type DeadLetterQueueConfig struct {
	TargetArn       string
	MaxReceiveCount int32
}

// Validate validates the queue configuration
func (q *Queue) Validate() error {
	if q.Name == "" {
		return ErrInvalidQueueName
	}

	// FIFO queues must end with .fifo
	if q.FifoQueue && len(q.Name) < 5 {
		return errors.New("FIFO queue name must end with .fifo")
	}

	// Set defaults
	if q.MaximumMessageSize == 0 {
		q.MaximumMessageSize = 262144 // 256 KB
	}
	if q.MessageRetentionPeriod == 0 {
		q.MessageRetentionPeriod = 345600 // 4 days
	}
	if q.VisibilityTimeout == 0 {
		q.VisibilityTimeout = 30
	}

	// Validate ranges
	if q.DelaySeconds < 0 || q.DelaySeconds > 900 {
		return errors.New("delay seconds must be between 0 and 900")
	}
	if q.MaximumMessageSize < 1024 || q.MaximumMessageSize > 262144 {
		return errors.New("maximum message size must be between 1024 and 262144")
	}
	if q.MessageRetentionPeriod < 60 || q.MessageRetentionPeriod > 1209600 {
		return errors.New("message retention period must be between 60 and 1209600")
	}
	if q.VisibilityTimeout < 0 || q.VisibilityTimeout > 43200 {
		return errors.New("visibility timeout must be between 0 and 43200")
	}
	if q.ReceiveMessageWaitTimeSeconds < 0 || q.ReceiveMessageWaitTimeSeconds > 20 {
		return errors.New("receive message wait time must be between 0 and 20")
	}

	return nil
}

// IsReady returns true if the queue has a URL
func (q *Queue) IsReady() bool {
	return q.URL != ""
}
