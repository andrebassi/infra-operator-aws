package sns

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/sns"
	"infra-operator/internal/ports"
)

// TopicUseCase implements the SNS use case
type TopicUseCase struct {
	repo ports.SNSRepository
}

// NewTopicUseCase creates a new SNS topic use case
func NewTopicUseCase(repo ports.SNSRepository) ports.SNSUseCase {
	return &TopicUseCase{
		repo: repo,
	}
}

// SyncTopic synchronizes the desired topic state with AWS
func (uc *TopicUseCase) SyncTopic(ctx context.Context, topic *sns.Topic) error {
	// Validate topic configuration
	if err := topic.Validate(); err != nil {
		return fmt.Errorf("invalid topic configuration: %w", err)
	}

	// Check if topic exists
	exists, err := uc.repo.Exists(ctx, topic.Name)
	if err != nil {
		return fmt.Errorf("failed to check topic existence: %w", err)
	}

	if !exists {
		// Create new topic
		if err := uc.repo.Create(ctx, topic); err != nil {
			return fmt.Errorf("failed to create topic: %w", err)
		}

		// Get topic details after creation
		created, err := uc.repo.GetByName(ctx, topic.Name)
		if err != nil {
			return fmt.Errorf("failed to get created topic: %w", err)
		}

		// Update topic with AWS-provided values
		topic.ARN = created.ARN
		topic.SubscriptionsConfirmed = created.SubscriptionsConfirmed
		topic.SubscriptionsPending = created.SubscriptionsPending
		topic.SubscriptionsDeleted = created.SubscriptionsDeleted

		// Create subscriptions
		if err := uc.syncSubscriptions(ctx, topic); err != nil {
			return fmt.Errorf("failed to sync subscriptions: %w", err)
		}

		return nil
	}

	// Topic exists, get current state
	current, err := uc.repo.GetByName(ctx, topic.Name)
	if err != nil {
		return fmt.Errorf("failed to get topic: %w", err)
	}

	// Update topic with AWS values
	topic.ARN = current.ARN

	// Check if update is needed
	if uc.needsUpdate(topic, current) {
		if err := uc.repo.Update(ctx, topic); err != nil {
			return fmt.Errorf("failed to update topic: %w", err)
		}

		// Get updated state
		updated, err := uc.repo.GetByName(ctx, topic.Name)
		if err != nil {
			return fmt.Errorf("failed to get updated topic: %w", err)
		}
		current = updated
	}

	// Update tags if needed
	if !uc.tagsEqual(topic.Tags, current.Tags) {
		if err := uc.repo.TagTopic(ctx, topic.ARN, topic.Tags); err != nil {
			return fmt.Errorf("failed to update tags: %w", err)
		}
	}

	// Sync subscriptions
	if err := uc.syncSubscriptions(ctx, topic); err != nil {
		return fmt.Errorf("failed to sync subscriptions: %w", err)
	}

	// Update metrics
	topic.SubscriptionsConfirmed = current.SubscriptionsConfirmed
	topic.SubscriptionsPending = current.SubscriptionsPending
	topic.SubscriptionsDeleted = current.SubscriptionsDeleted

	return nil
}

// DeleteTopic deletes a topic
func (uc *TopicUseCase) DeleteTopic(ctx context.Context, topic *sns.Topic) error {
	if topic.ARN == "" {
		// Topic ARN not set, try to get it by name
		current, err := uc.repo.GetByName(ctx, topic.Name)
		if err != nil {
			// Topic doesn't exist, nothing to delete
			return nil
		}
		topic.ARN = current.ARN
	}

	if err := uc.repo.Delete(ctx, topic.ARN); err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	return nil
}

// syncSubscriptions synchronizes subscriptions
func (uc *TopicUseCase) syncSubscriptions(ctx context.Context, topic *sns.Topic) error {
	// Get current subscriptions
	currentSubs, err := uc.repo.ListSubscriptions(ctx, topic.ARN)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	// Create map of desired subscriptions by endpoint
	desiredMap := make(map[string]*sns.Subscription)
	for i := range topic.Subscriptions {
		key := topic.Subscriptions[i].Protocol + ":" + topic.Subscriptions[i].Endpoint
		desiredMap[key] = &topic.Subscriptions[i]
	}

	// Create map of current subscriptions by endpoint
	currentMap := make(map[string]sns.Subscription)
	for _, sub := range currentSubs {
		key := sub.Protocol + ":" + sub.Endpoint
		currentMap[key] = sub
	}

	// Add missing subscriptions
	for key, desiredSub := range desiredMap {
		if _, exists := currentMap[key]; !exists {
			if err := uc.repo.Subscribe(ctx, topic.ARN, desiredSub); err != nil {
				return fmt.Errorf("failed to create subscription %s: %w", key, err)
			}
		}
	}

	// Remove extra subscriptions (if any)
	for key, currentSub := range currentMap {
		if _, desired := desiredMap[key]; !desired {
			if err := uc.repo.Unsubscribe(ctx, currentSub.ARN); err != nil {
				return fmt.Errorf("failed to remove subscription %s: %w", key, err)
			}
		}
	}

	return nil
}

// needsUpdate checks if topic attributes need to be updated
func (uc *TopicUseCase) needsUpdate(desired, current *sns.Topic) bool {
	if desired.DisplayName != current.DisplayName {
		return true
	}
	if desired.DeliveryPolicy != current.DeliveryPolicy {
		return true
	}
	if desired.KMSMasterKeyID != current.KMSMasterKeyID {
		return true
	}
	return false
}

// tagsEqual compares tag maps
func (uc *TopicUseCase) tagsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
