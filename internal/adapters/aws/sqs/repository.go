package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"infra-operator/internal/domain/sqs"
	"infra-operator/internal/ports"
)

// Repository implements the SQS repository using AWS SDK
type Repository struct {
	client *awssqs.Client
}

// NewRepository creates a new SQS repository
func NewRepository(awsConfig aws.Config) ports.SQSRepository {
	var options []func(*awssqs.Options)

	// Support LocalStack endpoint override
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		options = append(options, func(o *awssqs.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	return &Repository{
		client: awssqs.NewFromConfig(awsConfig, options...),
	}
}

// Exists checks if a queue with the given name exists
func (r *Repository) Exists(ctx context.Context, queueName string) (bool, error) {
	input := &awssqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	}

	_, err := r.client.GetQueueUrl(ctx, input)
	if err != nil {
		// Queue not found
		return false, nil
	}

	return true, nil
}

// Create creates a new SQS queue
func (r *Repository) Create(ctx context.Context, queue *sqs.Queue) error {
	attributes := make(map[string]string)

	// Basic attributes
	if queue.DelaySeconds > 0 {
		attributes["DelaySeconds"] = strconv.Itoa(int(queue.DelaySeconds))
	}
	if queue.MaximumMessageSize > 0 {
		attributes["MaximumMessageSize"] = strconv.Itoa(int(queue.MaximumMessageSize))
	}
	if queue.MessageRetentionPeriod > 0 {
		attributes["MessageRetentionPeriod"] = strconv.Itoa(int(queue.MessageRetentionPeriod))
	}
	if queue.VisibilityTimeout > 0 {
		attributes["VisibilityTimeout"] = strconv.Itoa(int(queue.VisibilityTimeout))
	}
	if queue.ReceiveMessageWaitTimeSeconds > 0 {
		attributes["ReceiveMessageWaitTimeSeconds"] = strconv.Itoa(int(queue.ReceiveMessageWaitTimeSeconds))
	}

	// FIFO attributes
	if queue.FifoQueue {
		attributes["FifoQueue"] = "true"
		if queue.ContentBasedDeduplication {
			attributes["ContentBasedDeduplication"] = "true"
		}
	}

	// Dead Letter Queue
	if queue.DeadLetterQueue != nil {
		redrivePolicy := map[string]interface{}{
			"deadLetterTargetArn": queue.DeadLetterQueue.TargetArn,
			"maxReceiveCount":     queue.DeadLetterQueue.MaxReceiveCount,
		}
		redrivePolicyJSON, err := json.Marshal(redrivePolicy)
		if err != nil {
			return fmt.Errorf("failed to marshal redrive policy: %w", err)
		}
		attributes["RedrivePolicy"] = string(redrivePolicyJSON)
	}

	// KMS Encryption
	if queue.KMSMasterKeyID != "" {
		attributes["KmsMasterKeyId"] = queue.KMSMasterKeyID
		if queue.KMSDataKeyReusePeriodSeconds > 0 {
			attributes["KmsDataKeyReusePeriodSeconds"] = strconv.Itoa(int(queue.KMSDataKeyReusePeriodSeconds))
		}
	}

	input := &awssqs.CreateQueueInput{
		QueueName:  aws.String(queue.Name),
		Attributes: attributes,
		Tags:       queue.Tags,
	}

	output, err := r.client.CreateQueue(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create queue: %w", err)
	}

	queue.URL = aws.ToString(output.QueueUrl)
	return nil
}

// Get retrieves queue details by URL
func (r *Repository) Get(ctx context.Context, queueURL string) (*sqs.Queue, error) {
	// Get queue attributes
	attrInput := &awssqs.GetQueueAttributesInput{
		QueueUrl: aws.String(queueURL),
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameAll,
		},
	}

	attrOutput, err := r.client.GetQueueAttributes(ctx, attrInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue attributes: %w", err)
	}

	queue := &sqs.Queue{
		URL: queueURL,
	}

	// Parse attributes
	attrs := attrOutput.Attributes
	if queueArn, ok := attrs["QueueArn"]; ok {
		queue.ARN = queueArn
	}
	if delaySeconds, ok := attrs["DelaySeconds"]; ok {
		val, _ := strconv.ParseInt(delaySeconds, 10, 32)
		queue.DelaySeconds = int32(val)
	}
	if maxMessageSize, ok := attrs["MaximumMessageSize"]; ok {
		val, _ := strconv.ParseInt(maxMessageSize, 10, 32)
		queue.MaximumMessageSize = int32(val)
	}
	if retention, ok := attrs["MessageRetentionPeriod"]; ok {
		val, _ := strconv.ParseInt(retention, 10, 32)
		queue.MessageRetentionPeriod = int32(val)
	}
	if visibility, ok := attrs["VisibilityTimeout"]; ok {
		val, _ := strconv.ParseInt(visibility, 10, 32)
		queue.VisibilityTimeout = int32(val)
	}
	if waitTime, ok := attrs["ReceiveMessageWaitTimeSeconds"]; ok {
		val, _ := strconv.ParseInt(waitTime, 10, 32)
		queue.ReceiveMessageWaitTimeSeconds = int32(val)
	}
	if fifo, ok := attrs["FifoQueue"]; ok {
		queue.FifoQueue = fifo == "true"
	}
	if dedup, ok := attrs["ContentBasedDeduplication"]; ok {
		queue.ContentBasedDeduplication = dedup == "true"
	}
	if approxMessages, ok := attrs["ApproximateNumberOfMessages"]; ok {
		val, _ := strconv.ParseInt(approxMessages, 10, 64)
		queue.ApproximateNumberOfMessages = val
	}
	if approxNotVisible, ok := attrs["ApproximateNumberOfMessagesNotVisible"]; ok {
		val, _ := strconv.ParseInt(approxNotVisible, 10, 64)
		queue.ApproximateNumberOfMessagesNotVisible = val
	}

	// Parse KMS
	if kmsKeyId, ok := attrs["KmsMasterKeyId"]; ok {
		queue.KMSMasterKeyID = kmsKeyId
	}
	if kmsReuse, ok := attrs["KmsDataKeyReusePeriodSeconds"]; ok {
		val, _ := strconv.ParseInt(kmsReuse, 10, 32)
		queue.KMSDataKeyReusePeriodSeconds = int32(val)
	}

	// Parse Redrive Policy (DLQ)
	if redrivePolicy, ok := attrs["RedrivePolicy"]; ok {
		var policy map[string]interface{}
		if err := json.Unmarshal([]byte(redrivePolicy), &policy); err == nil {
			if targetArn, ok := policy["deadLetterTargetArn"].(string); ok {
				maxReceive := int32(0)
				if count, ok := policy["maxReceiveCount"].(float64); ok {
					maxReceive = int32(count)
				}
				queue.DeadLetterQueue = &sqs.DeadLetterQueueConfig{
					TargetArn:       targetArn,
					MaxReceiveCount: maxReceive,
				}
			}
		}
	}

	return queue, nil
}

// GetByName retrieves queue details by name
func (r *Repository) GetByName(ctx context.Context, queueName string) (*sqs.Queue, error) {
	// Get queue URL first
	urlInput := &awssqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	}

	urlOutput, err := r.client.GetQueueUrl(ctx, urlInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue URL: %w", err)
	}

	return r.Get(ctx, aws.ToString(urlOutput.QueueUrl))
}

// Update updates queue attributes
func (r *Repository) Update(ctx context.Context, queue *sqs.Queue) error {
	attributes := make(map[string]string)

	// Update mutable attributes
	if queue.DelaySeconds >= 0 {
		attributes["DelaySeconds"] = strconv.Itoa(int(queue.DelaySeconds))
	}
	if queue.MaximumMessageSize > 0 {
		attributes["MaximumMessageSize"] = strconv.Itoa(int(queue.MaximumMessageSize))
	}
	if queue.MessageRetentionPeriod > 0 {
		attributes["MessageRetentionPeriod"] = strconv.Itoa(int(queue.MessageRetentionPeriod))
	}
	if queue.VisibilityTimeout >= 0 {
		attributes["VisibilityTimeout"] = strconv.Itoa(int(queue.VisibilityTimeout))
	}
	if queue.ReceiveMessageWaitTimeSeconds >= 0 {
		attributes["ReceiveMessageWaitTimeSeconds"] = strconv.Itoa(int(queue.ReceiveMessageWaitTimeSeconds))
	}

	// Update DLQ if present
	if queue.DeadLetterQueue != nil {
		redrivePolicy := map[string]interface{}{
			"deadLetterTargetArn": queue.DeadLetterQueue.TargetArn,
			"maxReceiveCount":     queue.DeadLetterQueue.MaxReceiveCount,
		}
		redrivePolicyJSON, err := json.Marshal(redrivePolicy)
		if err != nil {
			return fmt.Errorf("failed to marshal redrive policy: %w", err)
		}
		attributes["RedrivePolicy"] = string(redrivePolicyJSON)
	}

	// Update KMS if present
	if queue.KMSMasterKeyID != "" {
		attributes["KmsMasterKeyId"] = queue.KMSMasterKeyID
		if queue.KMSDataKeyReusePeriodSeconds > 0 {
			attributes["KmsDataKeyReusePeriodSeconds"] = strconv.Itoa(int(queue.KMSDataKeyReusePeriodSeconds))
		}
	}

	if len(attributes) > 0 {
		input := &awssqs.SetQueueAttributesInput{
			QueueUrl:   aws.String(queue.URL),
			Attributes: attributes,
		}

		_, err := r.client.SetQueueAttributes(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to update queue attributes: %w", err)
		}
	}

	return nil
}

// Delete deletes a queue
func (r *Repository) Delete(ctx context.Context, queueURL string) error {
	input := &awssqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	}

	_, err := r.client.DeleteQueue(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete queue: %w", err)
	}

	return nil
}

// TagQueue tags a queue
func (r *Repository) TagQueue(ctx context.Context, queueURL string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	input := &awssqs.TagQueueInput{
		QueueUrl: aws.String(queueURL),
		Tags:     tags,
	}

	_, err := r.client.TagQueue(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to tag queue: %w", err)
	}

	return nil
}
