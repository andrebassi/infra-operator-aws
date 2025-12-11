package dynamodb

import (
	"errors"
	"time"
)

var (
	ErrTableNotFound    = errors.New("table not found")
	ErrInvalidTableName = errors.New("invalid table name")
	ErrInvalidKey       = errors.New("invalid key configuration")
)

// Table represents a DynamoDB table in the domain
type Table struct {
	Name                   string
	ARN                    string
	HashKey                AttributeDefinition
	RangeKey               *AttributeDefinition
	Attributes             []AttributeDefinition
	BillingMode            string
	GlobalSecondaryIndexes []GlobalSecondaryIndex
	StreamEnabled          bool
	StreamViewType         string
	StreamARN              string
	PointInTimeRecovery    bool
	Tags                   map[string]string
	Status                 string
	ItemCount              int64
	TableSizeBytes         int64
	CreationTime           *time.Time
	LastSyncTime           *time.Time
	DeletionPolicy         string
}

// AttributeDefinition represents a DynamoDB attribute
type AttributeDefinition struct {
	Name string
	Type string // S, N, B
}

// GlobalSecondaryIndex represents a GSI
type GlobalSecondaryIndex struct {
	IndexName        string
	HashKey          string
	RangeKey         string
	ProjectionType   string
	NonKeyAttributes []string
}

// Validate validates the table configuration
func (t *Table) Validate() error {
	if t.Name == "" {
		return ErrInvalidTableName
	}

	if t.HashKey.Name == "" || t.HashKey.Type == "" {
		return ErrInvalidKey
	}

	// Validate attribute types
	validTypes := map[string]bool{"S": true, "N": true, "B": true}
	if !validTypes[t.HashKey.Type] {
		return ErrInvalidKey
	}

	if t.RangeKey != nil {
		if t.RangeKey.Name == "" || !validTypes[t.RangeKey.Type] {
			return ErrInvalidKey
		}
	}

	// Validate billing mode
	if t.BillingMode == "" {
		t.BillingMode = "PAY_PER_REQUEST"
	}
	if t.BillingMode != "PAY_PER_REQUEST" && t.BillingMode != "PROVISIONED" {
		return errors.New("invalid billing mode")
	}

	// Validate stream config
	if t.StreamEnabled && t.StreamViewType == "" {
		t.StreamViewType = "NEW_AND_OLD_IMAGES"
	}

	return nil
}

// IsReady returns true if the table is in ACTIVE status
func (t *Table) IsReady() bool {
	return t.Status == "ACTIVE"
}
