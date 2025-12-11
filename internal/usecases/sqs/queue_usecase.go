package sqs

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/sqs"
	"infra-operator/internal/ports"
)

// QueueUseCase implements the SQS use case
type QueueUseCase struct {
	repo ports.SQSRepository
}

// NewQueueUseCase creates a new SQS queue use case
func NewQueueUseCase(repo ports.SQSRepository) ports.SQSUseCase {
	return &QueueUseCase{
		repo: repo,
	}
}

// SyncQueue synchronizes the desired queue state with AWS
func (uc *QueueUseCase) SyncQueue(ctx context.Context, queue *sqs.Queue) error {
	// Validate queue configuration
	if err := queue.Validate(); err != nil {
		return fmt.Errorf("invalid queue configuration: %w", err)
	}

	// Check if queue exists
	exists, err := uc.repo.Exists(ctx, queue.Name)
	if err != nil {
		return fmt.Errorf("failed to check queue existence: %w", err)
	}

	if !exists {
		// Create new queue
		if err := uc.repo.Create(ctx, queue); err != nil {
			return fmt.Errorf("failed to create queue: %w", err)
		}

		// Get queue details after creation
		created, err := uc.repo.GetByName(ctx, queue.Name)
		if err != nil {
			return fmt.Errorf("failed to get created queue: %w", err)
		}

		// Update queue with AWS-provided values
		queue.URL = created.URL
		queue.ARN = created.ARN
		queue.ApproximateNumberOfMessages = created.ApproximateNumberOfMessages
		queue.ApproximateNumberOfMessagesNotVisible = created.ApproximateNumberOfMessagesNotVisible

		return nil
	}

	// Queue exists, get current state
	current, err := uc.repo.GetByName(ctx, queue.Name)
	if err != nil {
		return fmt.Errorf("failed to get queue: %w", err)
	}

	// Update queue with AWS values
	queue.URL = current.URL
	queue.ARN = current.ARN

	// Check if update is needed (compare mutable attributes)
	if uc.needsUpdate(queue, current) {
		if err := uc.repo.Update(ctx, queue); err != nil {
			return fmt.Errorf("failed to update queue: %w", err)
		}

		// Get updated state
		updated, err := uc.repo.GetByName(ctx, queue.Name)
		if err != nil {
			return fmt.Errorf("failed to get updated queue: %w", err)
		}
		current = updated
	}

	// Update tags if needed
	if !uc.tagsEqual(queue.Tags, current.Tags) {
		if err := uc.repo.TagQueue(ctx, queue.URL, queue.Tags); err != nil {
			return fmt.Errorf("failed to update tags: %w", err)
		}
	}

	// Update metrics
	queue.ApproximateNumberOfMessages = current.ApproximateNumberOfMessages
	queue.ApproximateNumberOfMessagesNotVisible = current.ApproximateNumberOfMessagesNotVisible

	return nil
}

// DeleteQueue deletes a queue
func (uc *QueueUseCase) DeleteQueue(ctx context.Context, queue *sqs.Queue) error {
	if queue.URL == "" {
		// Queue URL not set, try to get it by name
		current, err := uc.repo.GetByName(ctx, queue.Name)
		if err != nil {
			// Queue doesn't exist, nothing to delete
			return nil
		}
		queue.URL = current.URL
	}

	if err := uc.repo.Delete(ctx, queue.URL); err != nil {
		return fmt.Errorf("failed to delete queue: %w", err)
	}

	return nil
}

// needsUpdate checks if queue attributes need to be updated
func (uc *QueueUseCase) needsUpdate(desired, current *sqs.Queue) bool {
	// Compare mutable attributes
	if desired.DelaySeconds != current.DelaySeconds {
		return true
	}
	if desired.MaximumMessageSize != current.MaximumMessageSize {
		return true
	}
	if desired.MessageRetentionPeriod != current.MessageRetentionPeriod {
		return true
	}
	if desired.VisibilityTimeout != current.VisibilityTimeout {
		return true
	}
	if desired.ReceiveMessageWaitTimeSeconds != current.ReceiveMessageWaitTimeSeconds {
		return true
	}

	// Compare DLQ configuration
	if !uc.dlqEqual(desired.DeadLetterQueue, current.DeadLetterQueue) {
		return true
	}

	// Compare KMS configuration
	if desired.KMSMasterKeyID != current.KMSMasterKeyID {
		return true
	}
	if desired.KMSDataKeyReusePeriodSeconds != current.KMSDataKeyReusePeriodSeconds {
		return true
	}

	return false
}

// dlqEqual compares DLQ configurations
func (uc *QueueUseCase) dlqEqual(a, b *sqs.DeadLetterQueueConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.TargetArn == b.TargetArn && a.MaxReceiveCount == b.MaxReceiveCount
}

// tagsEqual compares tag maps
func (uc *QueueUseCase) tagsEqual(a, b map[string]string) bool {
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
