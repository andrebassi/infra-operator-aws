package sns

import (
	"errors"
	"time"
)

var (
	ErrTopicNotFound    = errors.New("topic not found")
	ErrInvalidTopicName = errors.New("invalid topic name")
)

// Topic represents an SNS topic in the domain
type Topic struct {
	Name                      string
	ARN                       string
	DisplayName               string
	FifoTopic                 bool
	ContentBasedDeduplication bool
	KMSMasterKeyID            string
	DeliveryPolicy            string
	Subscriptions             []Subscription
	Tags                      map[string]string
	SubscriptionsConfirmed    int32
	SubscriptionsPending      int32
	SubscriptionsDeleted      int32
	CreationTime              *time.Time
	LastSyncTime              *time.Time
	DeletionPolicy            string
}

// Subscription represents a subscription to the topic
type Subscription struct {
	ARN                string
	Protocol           string
	Endpoint           string
	FilterPolicy       string
	RawMessageDelivery bool
	DeadLetterQueueArn string
	Confirmed          bool
}

// Validate validates the topic configuration
func (t *Topic) Validate() error {
	if t.Name == "" {
		return ErrInvalidTopicName
	}

	// FIFO topics must end with .fifo
	if t.FifoTopic && len(t.Name) < 5 {
		return errors.New("FIFO topic name must end with .fifo")
	}

	// Content-based deduplication only valid for FIFO
	if t.ContentBasedDeduplication && !t.FifoTopic {
		return errors.New("content-based deduplication requires FIFO topic")
	}

	// Validate subscriptions
	for i, sub := range t.Subscriptions {
		if sub.Protocol == "" {
			return errors.New("subscription protocol is required")
		}
		if sub.Endpoint == "" {
			return errors.New("subscription endpoint is required")
		}

		// Validate protocol
		validProtocols := map[string]bool{
			"http":        true,
			"https":       true,
			"email":       true,
			"email-json":  true,
			"sms":         true,
			"sqs":         true,
			"lambda":      true,
			"application": true,
			"firehose":    true,
		}
		if !validProtocols[sub.Protocol] {
			return errors.New("invalid subscription protocol at index " + string(rune(i)))
		}
	}

	return nil
}

// IsReady returns true if the topic has an ARN
func (t *Topic) IsReady() bool {
	return t.ARN != ""
}
