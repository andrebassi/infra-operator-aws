package lambda

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"infra-operator/internal/domain/lambda"
	"infra-operator/internal/ports"
)

// Repository implements the LambdaRepository interface using AWS SDK
type Repository struct {
	client *awslambda.Client
}

// NewRepository creates a new Lambda repository
func NewRepository(awsConfig aws.Config) ports.LambdaRepository {
	var options []func(*awslambda.Options)

	// Support LocalStack endpoint
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		options = append(options, func(o *awslambda.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	return &Repository{
		client: awslambda.NewFromConfig(awsConfig, options...),
	}
}

// Exists checks if a Lambda function exists
func (r *Repository) Exists(ctx context.Context, functionName string) (bool, error) {
	_, err := r.client.GetFunction(ctx, &awslambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})

	if err != nil {
		// Check if it's a ResourceNotFoundException
		var notFoundErr *types.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if function exists: %w", err)
	}

	return true, nil
}

// Create creates a new Lambda function
func (r *Repository) Create(ctx context.Context, function *lambda.Function) error {
	// Prepare function code
	code, err := r.prepareFunctionCode(function.Code)
	if err != nil {
		return fmt.Errorf("failed to prepare function code: %w", err)
	}

	// Prepare environment variables
	var environment *types.Environment
	if len(function.Environment) > 0 {
		environment = &types.Environment{
			Variables: function.Environment,
		}
	}

	// Prepare VPC config
	var vpcConfig *types.VpcConfig
	if function.VpcConfig != nil {
		vpcConfig = &types.VpcConfig{
			SecurityGroupIds: function.VpcConfig.SecurityGroupIds,
			SubnetIds:        function.VpcConfig.SubnetIds,
		}
	}

	// Build create function input
	input := &awslambda.CreateFunctionInput{
		FunctionName: aws.String(function.Name),
		Runtime:      types.Runtime(function.Runtime),
		Handler:      aws.String(function.Handler),
		Role:         aws.String(function.Role),
		Code:         code,
		Description:  aws.String(function.Description),
		Timeout:      aws.Int32(function.Timeout),
		MemorySize:   aws.Int32(function.MemorySize),
		Environment:  environment,
		VpcConfig:    vpcConfig,
		Tags:         function.Tags,
	}

	// Add layers if specified
	if len(function.Layers) > 0 {
		input.Layers = function.Layers
	}

	output, err := r.client.CreateFunction(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create Lambda function: %w", err)
	}

	// Update function with created details
	function.ARN = aws.ToString(output.FunctionArn)
	function.State = string(output.State)
	function.StateReason = aws.ToString(output.StateReason)
	function.Version = aws.ToString(output.Version)
	function.CodeSize = output.CodeSize
	if output.LastModified != nil {
		if t, err := time.Parse(time.RFC3339, *output.LastModified); err == nil {
			function.LastModified = t
		}
	}

	return nil
}

// Get retrieves a Lambda function by name
func (r *Repository) Get(ctx context.Context, functionName string) (*lambda.Function, error) {
	output, err := r.client.GetFunction(ctx, &awslambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	return r.mapToFunction(output.Configuration, output.Tags), nil
}

// GetByARN retrieves a Lambda function by ARN
func (r *Repository) GetByARN(ctx context.Context, functionARN string) (*lambda.Function, error) {
	return r.Get(ctx, functionARN) // Lambda accepts both name and ARN
}

// UpdateConfiguration updates an existing Lambda function configuration
func (r *Repository) UpdateConfiguration(ctx context.Context, function *lambda.Function) error {
	// Prepare environment variables
	var environment *types.Environment
	if len(function.Environment) > 0 {
		environment = &types.Environment{
			Variables: function.Environment,
		}
	}

	// Prepare VPC config
	var vpcConfig *types.VpcConfig
	if function.VpcConfig != nil {
		vpcConfig = &types.VpcConfig{
			SecurityGroupIds: function.VpcConfig.SecurityGroupIds,
			SubnetIds:        function.VpcConfig.SubnetIds,
		}
	}

	input := &awslambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(function.Name),
		Runtime:      types.Runtime(function.Runtime),
		Handler:      aws.String(function.Handler),
		Role:         aws.String(function.Role),
		Description:  aws.String(function.Description),
		Timeout:      aws.Int32(function.Timeout),
		MemorySize:   aws.Int32(function.MemorySize),
		Environment:  environment,
		VpcConfig:    vpcConfig,
	}

	// Add layers if specified
	if len(function.Layers) > 0 {
		input.Layers = function.Layers
	}

	output, err := r.client.UpdateFunctionConfiguration(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update function configuration: %w", err)
	}

	// Update function state
	function.State = string(output.State)
	function.StateReason = aws.ToString(output.StateReason)
	function.Version = aws.ToString(output.Version)
	if output.LastModified != nil {
		if t, err := time.Parse(time.RFC3339, *output.LastModified); err == nil {
			function.LastModified = t
		}
	}

	return nil
}

// UpdateCode updates the Lambda function code
func (r *Repository) UpdateCode(ctx context.Context, function *lambda.Function) error {
	input := &awslambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(function.Name),
	}

	// Set code source based on what's provided
	if function.Code.ZipFile != "" {
		decoded, err := base64.StdEncoding.DecodeString(function.Code.ZipFile)
		if err != nil {
			return fmt.Errorf("failed to decode zip file: %w", err)
		}
		input.ZipFile = decoded
	} else if function.Code.S3Bucket != "" {
		input.S3Bucket = aws.String(function.Code.S3Bucket)
		input.S3Key = aws.String(function.Code.S3Key)
		if function.Code.S3ObjectVersion != "" {
			input.S3ObjectVersion = aws.String(function.Code.S3ObjectVersion)
		}
	} else if function.Code.ImageUri != "" {
		input.ImageUri = aws.String(function.Code.ImageUri)
	}

	output, err := r.client.UpdateFunctionCode(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update function code: %w", err)
	}

	// Update function details
	function.State = string(output.State)
	function.CodeSize = output.CodeSize
	if output.LastModified != nil {
		if t, err := time.Parse(time.RFC3339, *output.LastModified); err == nil {
			function.LastModified = t
		}
	}

	return nil
}

// Delete deletes a Lambda function
func (r *Repository) Delete(ctx context.Context, functionName string) error {
	_, err := r.client.DeleteFunction(ctx, &awslambda.DeleteFunctionInput{
		FunctionName: aws.String(functionName),
	})

	if err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
	}

	return nil
}

// TagFunction adds or updates tags on a Lambda function
func (r *Repository) TagFunction(ctx context.Context, functionARN string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	_, err := r.client.TagResource(ctx, &awslambda.TagResourceInput{
		Resource: aws.String(functionARN),
		Tags:     tags,
	})

	if err != nil {
		return fmt.Errorf("failed to tag function: %w", err)
	}

	return nil
}

// UntagFunction removes tags from a Lambda function
func (r *Repository) UntagFunction(ctx context.Context, functionARN string, tagKeys []string) error {
	if len(tagKeys) == 0 {
		return nil
	}

	_, err := r.client.UntagResource(ctx, &awslambda.UntagResourceInput{
		Resource: aws.String(functionARN),
		TagKeys:  tagKeys,
	})

	if err != nil {
		return fmt.Errorf("failed to untag function: %w", err)
	}

	return nil
}

// Helper functions

func (r *Repository) prepareFunctionCode(code lambda.Code) (*types.FunctionCode, error) {
	functionCode := &types.FunctionCode{}

	if code.ZipFile != "" {
		decoded, err := base64.StdEncoding.DecodeString(code.ZipFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decode zip file: %w", err)
		}
		functionCode.ZipFile = decoded
	} else if code.S3Bucket != "" {
		functionCode.S3Bucket = aws.String(code.S3Bucket)
		functionCode.S3Key = aws.String(code.S3Key)
		if code.S3ObjectVersion != "" {
			functionCode.S3ObjectVersion = aws.String(code.S3ObjectVersion)
		}
	} else if code.ImageUri != "" {
		functionCode.ImageUri = aws.String(code.ImageUri)
	}

	return functionCode, nil
}

func (r *Repository) mapToFunction(config *types.FunctionConfiguration, tags map[string]string) *lambda.Function {
	function := &lambda.Function{
		Name:        aws.ToString(config.FunctionName),
		ARN:         aws.ToString(config.FunctionArn),
		Description: aws.ToString(config.Description),
		Runtime:     string(config.Runtime),
		Handler:     aws.ToString(config.Handler),
		Role:        aws.ToString(config.Role),
		Timeout:     aws.ToInt32(config.Timeout),
		MemorySize:  aws.ToInt32(config.MemorySize),
		State:       string(config.State),
		StateReason: aws.ToString(config.StateReason),
		Version:     aws.ToString(config.Version),
		CodeSize:    config.CodeSize,
		Tags:        tags,
	}

	// Map environment variables
	if config.Environment != nil {
		function.Environment = config.Environment.Variables
	}

	// Map VPC config
	if config.VpcConfig != nil {
		function.VpcConfig = &lambda.VpcConfig{
			SecurityGroupIds: config.VpcConfig.SecurityGroupIds,
			SubnetIds:        config.VpcConfig.SubnetIds,
		}
	}

	// Map layers
	if len(config.Layers) > 0 {
		function.Layers = make([]string, len(config.Layers))
		for i, layer := range config.Layers {
			function.Layers[i] = aws.ToString(layer.Arn)
		}
	}

	// Parse LastModified
	if config.LastModified != nil {
		if t, err := time.Parse(time.RFC3339, *config.LastModified); err == nil {
			function.LastModified = t
		}
	}

	return function
}
