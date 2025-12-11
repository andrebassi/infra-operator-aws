package sns

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"

	"infra-operator/internal/domain/sns"
	"infra-operator/internal/ports"
)

// Repository implements the SNS repository using AWS SDK
type Repository struct {
	client *awssns.Client
}

// NewRepository creates a new SNS repository
func NewRepository(awsConfig aws.Config) ports.SNSRepository {
	var options []func(*awssns.Options)

	// Support LocalStack endpoint override
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		options = append(options, func(o *awssns.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	return &Repository{
		client: awssns.NewFromConfig(awsConfig, options...),
	}
}

// Exists checks if a topic with the given name exists
func (r *Repository) Exists(ctx context.Context, topicName string) (bool, error) {
	// List all topics and search for the name
	input := &awssns.ListTopicsInput{}
	output, err := r.client.ListTopics(ctx, input)
	if err != nil {
		return false, fmt.Errorf("failed to list topics: %w", err)
	}

	for _, topic := range output.Topics {
		if topic.TopicArn != nil && contains(*topic.TopicArn, topicName) {
			return true, nil
		}
	}

	return false, nil
}

// Create creates a new SNS topic
func (r *Repository) Create(ctx context.Context, topic *sns.Topic) error {
	attributes := make(map[string]string)

	// Display name
	if topic.DisplayName != "" {
		attributes["DisplayName"] = topic.DisplayName
	}

	// FIFO configuration
	if topic.FifoTopic {
		attributes["FifoTopic"] = "true"
		if topic.ContentBasedDeduplication {
			attributes["ContentBasedDeduplication"] = "true"
		}
	}

	// KMS encryption
	if topic.KMSMasterKeyID != "" {
		attributes["KmsMasterKeyId"] = topic.KMSMasterKeyID
	}

	// Delivery policy
	if topic.DeliveryPolicy != "" {
		attributes["DeliveryPolicy"] = topic.DeliveryPolicy
	}

	input := &awssns.CreateTopicInput{
		Name:       aws.String(topic.Name),
		Attributes: attributes,
		Tags:       convertTags(topic.Tags),
	}

	output, err := r.client.CreateTopic(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	topic.ARN = aws.ToString(output.TopicArn)
	return nil
}

// Get retrieves topic details by ARN
func (r *Repository) Get(ctx context.Context, topicARN string) (*sns.Topic, error) {
	input := &awssns.GetTopicAttributesInput{
		TopicArn: aws.String(topicARN),
	}

	output, err := r.client.GetTopicAttributes(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic attributes: %w", err)
	}

	topic := &sns.Topic{
		ARN: topicARN,
	}

	// Parse attributes
	attrs := output.Attributes
	if displayName, ok := attrs["DisplayName"]; ok {
		topic.DisplayName = displayName
	}
	if fifo, ok := attrs["FifoTopic"]; ok {
		topic.FifoTopic = fifo == "true"
	}
	if dedup, ok := attrs["ContentBasedDeduplication"]; ok {
		topic.ContentBasedDeduplication = dedup == "true"
	}
	if kms, ok := attrs["KmsMasterKeyId"]; ok {
		topic.KMSMasterKeyID = kms
	}
	if policy, ok := attrs["DeliveryPolicy"]; ok {
		topic.DeliveryPolicy = policy
	}
	if confirmed, ok := attrs["SubscriptionsConfirmed"]; ok {
		val, _ := strconv.ParseInt(confirmed, 10, 32)
		topic.SubscriptionsConfirmed = int32(val)
	}
	if pending, ok := attrs["SubscriptionsPending"]; ok {
		val, _ := strconv.ParseInt(pending, 10, 32)
		topic.SubscriptionsPending = int32(val)
	}
	if deleted, ok := attrs["SubscriptionsDeleted"]; ok {
		val, _ := strconv.ParseInt(deleted, 10, 32)
		topic.SubscriptionsDeleted = int32(val)
	}

	return topic, nil
}

// GetByName retrieves topic details by name
func (r *Repository) GetByName(ctx context.Context, topicName string) (*sns.Topic, error) {
	// Find topic ARN by name
	listInput := &awssns.ListTopicsInput{}
	listOutput, err := r.client.ListTopics(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list topics: %w", err)
	}

	var topicARN string
	for _, topic := range listOutput.Topics {
		if topic.TopicArn != nil && contains(*topic.TopicArn, topicName) {
			topicARN = *topic.TopicArn
			break
		}
	}

	if topicARN == "" {
		return nil, sns.ErrTopicNotFound
	}

	return r.Get(ctx, topicARN)
}

// Update updates topic attributes
func (r *Repository) Update(ctx context.Context, topic *sns.Topic) error {
	// Update display name
	if topic.DisplayName != "" {
		err := r.setTopicAttribute(ctx, topic.ARN, "DisplayName", topic.DisplayName)
		if err != nil {
			return err
		}
	}

	// Update delivery policy
	if topic.DeliveryPolicy != "" {
		err := r.setTopicAttribute(ctx, topic.ARN, "DeliveryPolicy", topic.DeliveryPolicy)
		if err != nil {
			return err
		}
	}

	// Update KMS if needed (note: cannot be removed once set)
	if topic.KMSMasterKeyID != "" {
		err := r.setTopicAttribute(ctx, topic.ARN, "KmsMasterKeyId", topic.KMSMasterKeyID)
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete deletes a topic
func (r *Repository) Delete(ctx context.Context, topicARN string) error {
	input := &awssns.DeleteTopicInput{
		TopicArn: aws.String(topicARN),
	}

	_, err := r.client.DeleteTopic(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	return nil
}

// TagTopic tags a topic
func (r *Repository) TagTopic(ctx context.Context, topicARN string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	input := &awssns.TagResourceInput{
		ResourceArn: aws.String(topicARN),
		Tags:        convertTags(tags),
	}

	_, err := r.client.TagResource(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to tag topic: %w", err)
	}

	return nil
}

// Subscribe creates a subscription to the topic
func (r *Repository) Subscribe(ctx context.Context, topicARN string, subscription *sns.Subscription) error {
	attributes := make(map[string]string)

	// Raw message delivery (for SQS, HTTP, HTTPS)
	if subscription.RawMessageDelivery {
		attributes["RawMessageDelivery"] = "true"
	}

	// Filter policy
	if subscription.FilterPolicy != "" {
		attributes["FilterPolicy"] = subscription.FilterPolicy
	}

	// DLQ for subscription
	if subscription.DeadLetterQueueArn != "" {
		attributes["RedrivePolicy"] = fmt.Sprintf(`{"deadLetterTargetArn":"%s"}`, subscription.DeadLetterQueueArn)
	}

	input := &awssns.SubscribeInput{
		TopicArn:   aws.String(topicARN),
		Protocol:   aws.String(subscription.Protocol),
		Endpoint:   aws.String(subscription.Endpoint),
		Attributes: attributes,
	}

	output, err := r.client.Subscribe(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	subscription.ARN = aws.ToString(output.SubscriptionArn)
	return nil
}

// Unsubscribe removes a subscription
func (r *Repository) Unsubscribe(ctx context.Context, subscriptionARN string) error {
	input := &awssns.UnsubscribeInput{
		SubscriptionArn: aws.String(subscriptionARN),
	}

	_, err := r.client.Unsubscribe(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	return nil
}

// ListSubscriptions lists all subscriptions for a topic
func (r *Repository) ListSubscriptions(ctx context.Context, topicARN string) ([]sns.Subscription, error) {
	input := &awssns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(topicARN),
	}

	output, err := r.client.ListSubscriptionsByTopic(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	var subscriptions []sns.Subscription
	for _, sub := range output.Subscriptions {
		subscriptions = append(subscriptions, sns.Subscription{
			ARN:       aws.ToString(sub.SubscriptionArn),
			Protocol:  aws.ToString(sub.Protocol),
			Endpoint:  aws.ToString(sub.Endpoint),
			Confirmed: sub.SubscriptionArn != nil && *sub.SubscriptionArn != "PendingConfirmation",
		})
	}

	return subscriptions, nil
}

// Helper methods

func (r *Repository) setTopicAttribute(ctx context.Context, topicARN, name, value string) error {
	input := &awssns.SetTopicAttributesInput{
		TopicArn:       aws.String(topicARN),
		AttributeName:  aws.String(name),
		AttributeValue: aws.String(value),
	}

	_, err := r.client.SetTopicAttributes(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to set topic attribute %s: %w", name, err)
	}

	return nil
}

func convertTags(tags map[string]string) []types.Tag {
	var result []types.Tag
	for k, v := range tags {
		result = append(result, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}
