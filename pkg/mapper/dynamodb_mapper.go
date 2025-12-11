package mapper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/dynamodb"
)

// CRToDomainTable converts DynamoDBTable CR to domain model
func CRToDomainTable(cr *infrav1alpha1.DynamoDBTable) *dynamodb.Table {
	table := &dynamodb.Table{
		Name: cr.Spec.TableName,
		HashKey: dynamodb.AttributeDefinition{
			Name: cr.Spec.HashKey.Name,
			Type: cr.Spec.HashKey.Type,
		},
		BillingMode:         cr.Spec.BillingMode,
		StreamEnabled:       cr.Spec.StreamEnabled,
		StreamViewType:      cr.Spec.StreamViewType,
		PointInTimeRecovery: cr.Spec.PointInTimeRecovery,
		Tags:                cr.Spec.Tags,
		DeletionPolicy:      cr.Spec.DeletionPolicy,
	}

	// Range key
	if cr.Spec.RangeKey != nil {
		table.RangeKey = &dynamodb.AttributeDefinition{
			Name: cr.Spec.RangeKey.Name,
			Type: cr.Spec.RangeKey.Type,
		}
	}

	// Additional attributes
	for _, attr := range cr.Spec.Attributes {
		table.Attributes = append(table.Attributes, dynamodb.AttributeDefinition{
			Name: attr.Name,
			Type: attr.Type,
		})
	}

	// GSIs
	for _, gsi := range cr.Spec.GlobalSecondaryIndexes {
		table.GlobalSecondaryIndexes = append(table.GlobalSecondaryIndexes, dynamodb.GlobalSecondaryIndex{
			IndexName:        gsi.IndexName,
			HashKey:          gsi.HashKey,
			RangeKey:         gsi.RangeKey,
			ProjectionType:   gsi.ProjectionType,
			NonKeyAttributes: gsi.NonKeyAttributes,
		})
	}

	return table
}

// DomainTableToCRStatus updates CR status from domain model
func DomainTableToCRStatus(table *dynamodb.Table, cr *infrav1alpha1.DynamoDBTable) {
	cr.Status.TableARN = table.ARN
	cr.Status.TableStatus = table.Status
	cr.Status.ItemCount = table.ItemCount
	cr.Status.TableSizeBytes = table.TableSizeBytes
	cr.Status.StreamARN = table.StreamARN

	if table.LastSyncTime != nil {
		metaTime := metav1.Time{Time: *table.LastSyncTime}
		cr.Status.LastSyncTime = &metaTime
	}
}
