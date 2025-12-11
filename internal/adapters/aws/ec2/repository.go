package ec2

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"infra-operator/internal/domain/ec2"
	"infra-operator/internal/ports"
)

type Repository struct {
	client *awsec2.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{
		client: awsec2.NewFromConfig(cfg),
	}
}

func (r *Repository) Exists(ctx context.Context, instanceID string) (bool, error) {
	output, err := r.client.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		// If instance not found, return false
		return false, nil
	}
	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return false, nil
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, instance *ec2.Instance) error {
	input := &awsec2.RunInstancesInput{
		ImageId:      aws.String(instance.ImageID),
		InstanceType: types.InstanceType(instance.InstanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}

	if instance.KeyName != "" {
		input.KeyName = aws.String(instance.KeyName)
	}

	if instance.SubnetID != "" {
		input.SubnetId = aws.String(instance.SubnetID)
	}

	if len(instance.SecurityGroupIDs) > 0 {
		input.SecurityGroupIds = instance.SecurityGroupIDs
	}

	if instance.IAMInstanceProfile != "" {
		input.IamInstanceProfile = &types.IamInstanceProfileSpecification{
			Name: aws.String(instance.IAMInstanceProfile),
		}
	}

	if instance.UserData != "" {
		input.UserData = aws.String(base64.StdEncoding.EncodeToString([]byte(instance.UserData)))
	}

	if len(instance.BlockDeviceMappings) > 0 {
		input.BlockDeviceMappings = convertBlockDeviceMappings(instance.BlockDeviceMappings)
	}

	if instance.Monitoring {
		input.Monitoring = &types.RunInstancesMonitoringEnabled{
			Enabled: aws.Bool(true),
		}
	}

	if instance.DisableAPITermination {
		input.DisableApiTermination = aws.Bool(true)
	}

	if instance.EBSOptimized {
		input.EbsOptimized = aws.Bool(true)
	}

	if len(instance.Tags) > 0 {
		tagSpecs := []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         convertTags(instance.Tags),
			},
		}
		// Add Name tag
		if instance.InstanceName != "" {
			tagSpecs[0].Tags = append(tagSpecs[0].Tags, types.Tag{
				Key:   aws.String("Name"),
				Value: aws.String(instance.InstanceName),
			})
		}
		input.TagSpecifications = tagSpecs
	}

	output, err := r.client.RunInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if len(output.Instances) > 0 {
		inst := output.Instances[0]
		instance.InstanceID = aws.ToString(inst.InstanceId)
		instance.InstanceState = string(inst.State.Name)
		instance.PrivateIP = aws.ToString(inst.PrivateIpAddress)
		instance.PublicIP = aws.ToString(inst.PublicIpAddress)
		instance.PrivateDNS = aws.ToString(inst.PrivateDnsName)
		instance.PublicDNS = aws.ToString(inst.PublicDnsName)
		instance.AvailabilityZone = aws.ToString(inst.Placement.AvailabilityZone)
		if inst.LaunchTime != nil {
			t := *inst.LaunchTime
			instance.LaunchTime = &t
		}
	}

	return nil
}

func (r *Repository) Get(ctx context.Context, instanceID string) (*ec2.Instance, error) {
	output, err := r.client.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance not found")
	}

	inst := output.Reservations[0].Instances[0]

	instance := &ec2.Instance{
		InstanceID:       aws.ToString(inst.InstanceId),
		InstanceType:     string(inst.InstanceType),
		ImageID:          aws.ToString(inst.ImageId),
		KeyName:          aws.ToString(inst.KeyName),
		SubnetID:         aws.ToString(inst.SubnetId),
		InstanceState:    string(inst.State.Name),
		PrivateIP:        aws.ToString(inst.PrivateIpAddress),
		PublicIP:         aws.ToString(inst.PublicIpAddress),
		PrivateDNS:       aws.ToString(inst.PrivateDnsName),
		PublicDNS:        aws.ToString(inst.PublicDnsName),
		AvailabilityZone: aws.ToString(inst.Placement.AvailabilityZone),
	}

	if inst.LaunchTime != nil {
		t := *inst.LaunchTime
		instance.LaunchTime = &t
	}

	// Extract security groups
	for _, sg := range inst.SecurityGroups {
		instance.SecurityGroupIDs = append(instance.SecurityGroupIDs, aws.ToString(sg.GroupId))
	}

	// Extract IAM instance profile
	if inst.IamInstanceProfile != nil {
		instance.IAMInstanceProfile = aws.ToString(inst.IamInstanceProfile.Arn)
	}

	// Extract tags
	tags := make(map[string]string)
	for _, tag := range inst.Tags {
		if aws.ToString(tag.Key) != "Name" {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		} else {
			instance.InstanceName = aws.ToString(tag.Value)
		}
	}
	instance.Tags = tags

	return instance, nil
}

func (r *Repository) StartInstance(ctx context.Context, instanceID string) error {
	_, err := r.client.StartInstances(ctx, &awsec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}
	return nil
}

func (r *Repository) StopInstance(ctx context.Context, instanceID string) error {
	_, err := r.client.StopInstances(ctx, &awsec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}
	return nil
}

func (r *Repository) TerminateInstance(ctx context.Context, instanceID string) error {
	_, err := r.client.TerminateInstances(ctx, &awsec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}
	return nil
}

func (r *Repository) TagResource(ctx context.Context, instanceID string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	_, err := r.client.CreateTags(ctx, &awsec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      convertTags(tags),
	})
	if err != nil {
		return fmt.Errorf("failed to tag instance: %w", err)
	}
	return nil
}

func convertBlockDeviceMappings(mappings []ec2.BlockDeviceMapping) []types.BlockDeviceMapping {
	result := make([]types.BlockDeviceMapping, 0, len(mappings))
	for _, m := range mappings {
		bdm := types.BlockDeviceMapping{
			DeviceName: aws.String(m.DeviceName),
		}
		if m.EBS != nil {
			bdm.Ebs = &types.EbsBlockDevice{
				VolumeSize:          aws.Int32(m.EBS.VolumeSize),
				VolumeType:          types.VolumeType(m.EBS.VolumeType),
				DeleteOnTermination: aws.Bool(m.EBS.DeleteOnTermination),
				Encrypted:           aws.Bool(m.EBS.Encrypted),
			}
			if m.EBS.IOPS > 0 {
				bdm.Ebs.Iops = aws.Int32(m.EBS.IOPS)
			}
			if m.EBS.KMSKeyID != "" {
				bdm.Ebs.KmsKeyId = aws.String(m.EBS.KMSKeyID)
			}
		}
		result = append(result, bdm)
	}
	return result
}

func convertTags(tags map[string]string) []types.Tag {
	ec2Tags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		ec2Tags = append(ec2Tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return ec2Tags
}

// GetConsoleOutput obtém os logs do console da instância EC2
// Usa a API GetConsoleOutput da AWS que retorna os logs de boot/sistema
func (r *Repository) GetConsoleOutput(ctx context.Context, instanceID string, maxLines int) (*ports.ConsoleOutput, error) {
	output, err := r.client.GetConsoleOutput(ctx, &awsec2.GetConsoleOutputInput{
		InstanceId: aws.String(instanceID),
		Latest:     aws.Bool(true), // Obtém o output mais recente
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao obter console output: %w", err)
	}

	result := &ports.ConsoleOutput{
		Timestamp: time.Now(),
	}

	// O output pode estar vazio se a instância acabou de iniciar
	if output.Output == nil || *output.Output == "" {
		result.Output = "(aguardando logs do console...)"
		return result, nil
	}

	// Decodifica o output (pode estar em base64)
	rawOutput := aws.ToString(output.Output)

	// Tenta decodificar base64 se necessário
	if decoded, err := base64.StdEncoding.DecodeString(rawOutput); err == nil {
		rawOutput = string(decoded)
	}

	// Remove caracteres de controle ANSI/escape sequences para melhor legibilidade
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\[[0-9;]*[Hm]`)
	cleanOutput := ansiRegex.ReplaceAllString(rawOutput, "")

	// Limita o número de linhas se especificado
	if maxLines > 0 {
		lines := strings.Split(cleanOutput, "\n")
		if len(lines) > maxLines {
			// Pega as últimas N linhas
			lines = lines[len(lines)-maxLines:]
		}
		cleanOutput = strings.Join(lines, "\n")
	}

	result.Output = cleanOutput

	// Usa o timestamp da AWS se disponível
	if output.Timestamp != nil {
		result.Timestamp = *output.Timestamp
	}

	return result, nil
}
