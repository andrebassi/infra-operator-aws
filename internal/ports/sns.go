package ports

import (
	"context"
	"infra-operator/internal/domain/sns"
)

// SNSRepository defines the interface for SNS topic operations
type SNSRepository interface {
	// Exists checks if a topic with the given name exists
	Exists(ctx context.Context, topicName string) (bool, error)

	// Create creates a new SNS topic
	Create(ctx context.Context, topic *sns.Topic) error

	// Get retrieves topic details by ARN
	Get(ctx context.Context, topicARN string) (*sns.Topic, error)

	// GetByName retrieves topic details by name
	GetByName(ctx context.Context, topicName string) (*sns.Topic, error)

	// Update updates topic attributes
	Update(ctx context.Context, topic *sns.Topic) error

	// Delete deletes a topic
	Delete(ctx context.Context, topicARN string) error

	// TagTopic tags a topic
	TagTopic(ctx context.Context, topicARN string, tags map[string]string) error

	// Subscribe creates a subscription to the topic
	Subscribe(ctx context.Context, topicARN string, subscription *sns.Subscription) error

	// Unsubscribe removes a subscription
	Unsubscribe(ctx context.Context, subscriptionARN string) error

	// ListSubscriptions lists all subscriptions for a topic
	ListSubscriptions(ctx context.Context, topicARN string) ([]sns.Subscription, error)
}

// SNSUseCase defines the business logic interface for SNS operations
type SNSUseCase interface {
	// SyncTopic synchronizes the desired topic state with AWS
	SyncTopic(ctx context.Context, topic *sns.Topic) error

	// DeleteTopic deletes a topic
	DeleteTopic(ctx context.Context, topic *sns.Topic) error
}
