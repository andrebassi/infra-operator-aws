package rds

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsrds "github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	"infra-operator/internal/domain/rds"
	"infra-operator/internal/ports"
)

type Repository struct {
	client *awsrds.Client
}

func NewRepository(awsConfig aws.Config) ports.RDSRepository {
	var options []func(*awsrds.Options)
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		options = append(options, func(o *awsrds.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}
	return &Repository{
		client: awsrds.NewFromConfig(awsConfig, options...),
	}
}

func (r *Repository) Exists(ctx context.Context, dbInstanceIdentifier string) (bool, error) {
	_, err := r.client.DescribeDBInstances(ctx, &awsrds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstanceIdentifier),
	})
	if err != nil {
		var notFoundErr *types.DBInstanceNotFoundFault
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if DB instance exists: %w", err)
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, instance *rds.DBInstance) error {
	input := &awsrds.CreateDBInstanceInput{
		DBInstanceIdentifier:  aws.String(instance.DBInstanceIdentifier),
		Engine:                aws.String(instance.Engine),
		DBInstanceClass:       aws.String(instance.DBInstanceClass),
		AllocatedStorage:      aws.Int32(instance.AllocatedStorage),
		MasterUsername:        aws.String(instance.MasterUsername),
		MasterUserPassword:    aws.String(instance.MasterPassword),
		Port:                  aws.Int32(instance.Port),
		MultiAZ:               aws.Bool(instance.MultiAZ),
		PubliclyAccessible:    aws.Bool(instance.PubliclyAccessible),
		StorageEncrypted:      aws.Bool(instance.StorageEncrypted),
		BackupRetentionPeriod: aws.Int32(instance.BackupRetentionPeriod),
		Tags:                  convertTags(instance.Tags),
	}

	if instance.EngineVersion != "" {
		input.EngineVersion = aws.String(instance.EngineVersion)
	}
	if instance.DBName != "" {
		input.DBName = aws.String(instance.DBName)
	}
	if instance.PreferredBackupWindow != "" {
		input.PreferredBackupWindow = aws.String(instance.PreferredBackupWindow)
	}

	output, err := r.client.CreateDBInstance(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create DB instance: %w", err)
	}

	instance.DBInstanceArn = aws.ToString(output.DBInstance.DBInstanceArn)
	instance.Status = aws.ToString(output.DBInstance.DBInstanceStatus)
	if output.DBInstance.Endpoint != nil {
		instance.Endpoint = aws.ToString(output.DBInstance.Endpoint.Address)
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, dbInstanceIdentifier string) (*rds.DBInstance, error) {
	output, err := r.client.DescribeDBInstances(ctx, &awsrds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstanceIdentifier),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get DB instance: %w", err)
	}

	if len(output.DBInstances) == 0 {
		return nil, fmt.Errorf("DB instance not found")
	}

	return mapToDBInstance(&output.DBInstances[0]), nil
}

func (r *Repository) Update(ctx context.Context, instance *rds.DBInstance) error {
	input := &awsrds.ModifyDBInstanceInput{
		DBInstanceIdentifier:  aws.String(instance.DBInstanceIdentifier),
		AllocatedStorage:      aws.Int32(instance.AllocatedStorage),
		DBInstanceClass:       aws.String(instance.DBInstanceClass),
		BackupRetentionPeriod: aws.Int32(instance.BackupRetentionPeriod),
		ApplyImmediately:      aws.Bool(true),
	}

	if instance.PreferredBackupWindow != "" {
		input.PreferredBackupWindow = aws.String(instance.PreferredBackupWindow)
	}

	_, err := r.client.ModifyDBInstance(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update DB instance: %w", err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, dbInstanceIdentifier string, skipFinalSnapshot bool) error {
	input := &awsrds.DeleteDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbInstanceIdentifier),
		SkipFinalSnapshot:    aws.Bool(skipFinalSnapshot),
	}

	if !skipFinalSnapshot {
		input.FinalDBSnapshotIdentifier = aws.String(fmt.Sprintf("%s-final-snapshot-%d", dbInstanceIdentifier, time.Now().Unix()))
	}

	_, err := r.client.DeleteDBInstance(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete DB instance: %w", err)
	}

	return nil
}

func (r *Repository) TagResource(ctx context.Context, arn string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	_, err := r.client.AddTagsToResource(ctx, &awsrds.AddTagsToResourceInput{
		ResourceName: aws.String(arn),
		Tags:         convertTags(tags),
	})
	if err != nil {
		return fmt.Errorf("failed to tag resource: %w", err)
	}

	return nil
}

func convertTags(tags map[string]string) []types.Tag {
	var rdsTags []types.Tag
	for k, v := range tags {
		rdsTags = append(rdsTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return rdsTags
}

func mapToDBInstance(db *types.DBInstance) *rds.DBInstance {
	instance := &rds.DBInstance{
		DBInstanceIdentifier:  aws.ToString(db.DBInstanceIdentifier),
		DBInstanceArn:         aws.ToString(db.DBInstanceArn),
		Engine:                aws.ToString(db.Engine),
		EngineVersion:         aws.ToString(db.EngineVersion),
		DBInstanceClass:       aws.ToString(db.DBInstanceClass),
		AllocatedStorage:      aws.ToInt32(db.AllocatedStorage),
		MasterUsername:        aws.ToString(db.MasterUsername),
		DBName:                aws.ToString(db.DBName),
		Port:                  aws.ToInt32(db.DbInstancePort),
		MultiAZ:               aws.ToBool(db.MultiAZ),
		PubliclyAccessible:    aws.ToBool(db.PubliclyAccessible),
		StorageEncrypted:      aws.ToBool(db.StorageEncrypted),
		BackupRetentionPeriod: aws.ToInt32(db.BackupRetentionPeriod),
		PreferredBackupWindow: aws.ToString(db.PreferredBackupWindow),
		Status:                aws.ToString(db.DBInstanceStatus),
	}

	if db.Endpoint != nil {
		instance.Endpoint = aws.ToString(db.Endpoint.Address)
	}

	return instance
}
