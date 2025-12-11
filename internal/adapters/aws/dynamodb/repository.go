package dynamodb

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsddb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"infra-operator/internal/domain/dynamodb"
	"infra-operator/internal/ports"
)

// Repository implements ports.DynamoDBRepository using AWS SDK
type Repository struct {
	client *awsddb.Client
}

// NewRepository creates a new DynamoDB repository
func NewRepository(awsConfig aws.Config) ports.DynamoDBRepository {
	var options []func(*awsddb.Options)
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		options = append(options, func(o *awsddb.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	return &Repository{
		client: awsddb.NewFromConfig(awsConfig, options...),
	}
}

// Exists checks if table exists
func (r *Repository) Exists(ctx context.Context, tableName string) (bool, error) {
	_, err := r.client.DescribeTable(ctx, &awsddb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		// Check if error is ResourceNotFoundException
		return false, nil
	}

	return true, nil
}

// Create creates a new table
func (r *Repository) Create(ctx context.Context, table *dynamodb.Table) error {
	if err := table.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Build attribute definitions
	var attributes []types.AttributeDefinition
	attributes = append(attributes, types.AttributeDefinition{
		AttributeName: aws.String(table.HashKey.Name),
		AttributeType: types.ScalarAttributeType(table.HashKey.Type),
	})

	if table.RangeKey != nil {
		attributes = append(attributes, types.AttributeDefinition{
			AttributeName: aws.String(table.RangeKey.Name),
			AttributeType: types.ScalarAttributeType(table.RangeKey.Type),
		})
	}

	// Add attributes used in GSIs
	for _, attr := range table.Attributes {
		attributes = append(attributes, types.AttributeDefinition{
			AttributeName: aws.String(attr.Name),
			AttributeType: types.ScalarAttributeType(attr.Type),
		})
	}

	// Build key schema
	keySchema := []types.KeySchemaElement{
		{
			AttributeName: aws.String(table.HashKey.Name),
			KeyType:       types.KeyTypeHash,
		},
	}

	if table.RangeKey != nil {
		keySchema = append(keySchema, types.KeySchemaElement{
			AttributeName: aws.String(table.RangeKey.Name),
			KeyType:       types.KeyTypeRange,
		})
	}

	// Create table input
	input := &awsddb.CreateTableInput{
		TableName:            aws.String(table.Name),
		AttributeDefinitions: attributes,
		KeySchema:            keySchema,
		BillingMode:          types.BillingMode(table.BillingMode),
	}

	// Add GSIs if any
	if len(table.GlobalSecondaryIndexes) > 0 {
		var gsis []types.GlobalSecondaryIndex
		for _, gsi := range table.GlobalSecondaryIndexes {
			gsiKeySchema := []types.KeySchemaElement{
				{
					AttributeName: aws.String(gsi.HashKey),
					KeyType:       types.KeyTypeHash,
				},
			}

			if gsi.RangeKey != "" {
				gsiKeySchema = append(gsiKeySchema, types.KeySchemaElement{
					AttributeName: aws.String(gsi.RangeKey),
					KeyType:       types.KeyTypeRange,
				})
			}

			projection := &types.Projection{
				ProjectionType: types.ProjectionType(gsi.ProjectionType),
			}

			if gsi.ProjectionType == "INCLUDE" && len(gsi.NonKeyAttributes) > 0 {
				projection.NonKeyAttributes = gsi.NonKeyAttributes
			}

			gsis = append(gsis, types.GlobalSecondaryIndex{
				IndexName:  aws.String(gsi.IndexName),
				KeySchema:  gsiKeySchema,
				Projection: projection,
			})
		}
		input.GlobalSecondaryIndexes = gsis
	}

	// Add streaming if enabled
	if table.StreamEnabled {
		input.StreamSpecification = &types.StreamSpecification{
			StreamEnabled:  aws.Bool(true),
			StreamViewType: types.StreamViewType(table.StreamViewType),
		}
	}

	// Add tags
	if len(table.Tags) > 0 {
		var tags []types.Tag
		for k, v := range table.Tags {
			tags = append(tags, types.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}
		input.Tags = tags
	}

	_, err := r.client.CreateTable(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// Get retrieves table information
func (r *Repository) Get(ctx context.Context, tableName string) (*dynamodb.Table, error) {
	output, err := r.client.DescribeTable(ctx, &awsddb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		return nil, dynamodb.ErrTableNotFound
	}

	table := &dynamodb.Table{
		Name:   *output.Table.TableName,
		ARN:    *output.Table.TableArn,
		Status: string(output.Table.TableStatus),
	}

	// Parse key schema
	for _, key := range output.Table.KeySchema {
		for _, attr := range output.Table.AttributeDefinitions {
			if *attr.AttributeName == *key.AttributeName {
				attrDef := dynamodb.AttributeDefinition{
					Name: *attr.AttributeName,
					Type: string(attr.AttributeType),
				}

				if key.KeyType == types.KeyTypeHash {
					table.HashKey = attrDef
				} else if key.KeyType == types.KeyTypeRange {
					table.RangeKey = &attrDef
				}
			}
		}
	}

	// Get billing mode
	if output.Table.BillingModeSummary != nil {
		table.BillingMode = string(output.Table.BillingModeSummary.BillingMode)
	} else {
		table.BillingMode = "PROVISIONED"
	}

	// Get item count and size
	if output.Table.ItemCount != nil {
		table.ItemCount = *output.Table.ItemCount
	}
	if output.Table.TableSizeBytes != nil {
		table.TableSizeBytes = *output.Table.TableSizeBytes
	}

	// Get stream ARN
	if output.Table.LatestStreamArn != nil {
		table.StreamARN = *output.Table.LatestStreamArn
		table.StreamEnabled = true
	}

	return table, nil
}

// Update updates table configuration (placeholder)
func (r *Repository) Update(ctx context.Context, table *dynamodb.Table) error {
	// DynamoDB table updates are limited - mainly tags and streaming
	return nil
}

// Delete deletes a table
func (r *Repository) Delete(ctx context.Context, tableName string) error {
	_, err := r.client.DeleteTable(ctx, &awsddb.DeleteTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		return fmt.Errorf("failed to delete table: %w", err)
	}

	return nil
}

// UpdateTimeToLive enables/disables TTL
func (r *Repository) UpdateTimeToLive(ctx context.Context, tableName string, attributeName string, enabled bool) error {
	_, err := r.client.UpdateTimeToLive(ctx, &awsddb.UpdateTimeToLiveInput{
		TableName: aws.String(tableName),
		TimeToLiveSpecification: &types.TimeToLiveSpecification{
			Enabled:       aws.Bool(enabled),
			AttributeName: aws.String(attributeName),
		},
	})

	return err
}

// UpdateStreaming updates streaming configuration
func (r *Repository) UpdateStreaming(ctx context.Context, tableName string, enabled bool, viewType string) error {
	_, err := r.client.UpdateTable(ctx, &awsddb.UpdateTableInput{
		TableName: aws.String(tableName),
		StreamSpecification: &types.StreamSpecification{
			StreamEnabled:  aws.Bool(enabled),
			StreamViewType: types.StreamViewType(viewType),
		},
	})

	return err
}

// UpdatePointInTimeRecovery updates PITR configuration
func (r *Repository) UpdatePointInTimeRecovery(ctx context.Context, tableName string, enabled bool) error {
	_, err := r.client.UpdateContinuousBackups(ctx, &awsddb.UpdateContinuousBackupsInput{
		TableName: aws.String(tableName),
		PointInTimeRecoverySpecification: &types.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(enabled),
		},
	})

	return err
}

// TagResource applies tags to table
func (r *Repository) TagResource(ctx context.Context, tableARN string, tags map[string]string) error {
	var awsTags []types.Tag
	for k, v := range tags {
		awsTags = append(awsTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	_, err := r.client.TagResource(ctx, &awsddb.TagResourceInput{
		ResourceArn: aws.String(tableARN),
		Tags:        awsTags,
	})

	return err
}
