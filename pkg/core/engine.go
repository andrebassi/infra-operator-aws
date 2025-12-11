package core

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Engine é o motor principal que executa operações nos recursos AWS
type Engine struct {
	stateManager   *StateManager
	providerConfig *ProviderConfig
	awsConfig      aws.Config
	dryRun         bool
	verbose        bool
	output         OutputWriter
}

// OutputWriter interface para output customizado
type OutputWriter interface {
	Write(format string, args ...interface{})
	WriteVerbose(format string, args ...interface{})
}

// DefaultOutputWriter escreve para stdout
type DefaultOutputWriter struct {
	Verbose bool
}

func (w *DefaultOutputWriter) Write(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (w *DefaultOutputWriter) WriteVerbose(format string, args ...interface{}) {
	if w.Verbose {
		fmt.Printf(format+"\n", args...)
	}
}

// SilentOutputWriter não escreve nada (para API)
type SilentOutputWriter struct{}

func (w *SilentOutputWriter) Write(format string, args ...interface{})        {}
func (w *SilentOutputWriter) WriteVerbose(format string, args ...interface{}) {}

// EngineConfig configuração do engine
type EngineConfig struct {
	StateDir string
	Provider *ProviderConfig
	DryRun   bool
	Verbose  bool
	Output   OutputWriter
}

// NewEngine cria um novo engine
func NewEngine(ctx context.Context, cfg EngineConfig) (*Engine, error) {
	if cfg.Provider == nil {
		cfg.Provider = NewProviderConfigFromEnv()
	}

	awsCfg, err := cfg.Provider.GetAWSConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter configuração AWS: %w", err)
	}

	if cfg.Output == nil {
		cfg.Output = &DefaultOutputWriter{Verbose: cfg.Verbose}
	}

	return &Engine{
		stateManager:   NewStateManager(cfg.StateDir),
		providerConfig: cfg.Provider,
		awsConfig:      awsCfg,
		dryRun:         cfg.DryRun,
		verbose:        cfg.Verbose,
		output:         cfg.Output,
	}, nil
}

// Plan gera o plano de execução para os recursos
func (e *Engine) Plan(ctx context.Context, resources []Resource) (*PlanResult, error) {
	ordered := OrderByDependency(resources)
	result := &PlanResult{
		Resources: resources,
	}

	for _, r := range ordered {
		if r.Kind == "AWSProvider" {
			continue
		}

		existingState, _ := e.stateManager.LoadState(r.Kind, r.Metadata.Namespace, r.Metadata.Name)

		item := PlanItem{
			Kind:      r.Kind,
			Name:      r.Metadata.Name,
			Namespace: r.Metadata.Namespace,
		}

		if existingState == nil {
			item.Action = "create"
			result.ToCreate = append(result.ToCreate, item)
		} else {
			// TODO: Comparar specs para detectar mudanças
			item.Action = "no_change"
			result.NoChange = append(result.NoChange, item)
		}
	}

	return result, nil
}

// Apply aplica os recursos (cria ou atualiza)
func (e *Engine) Apply(ctx context.Context, resources []Resource) (*ApplyResult, error) {
	ordered := OrderByDependency(resources)
	result := &ApplyResult{}

	// Extrai providers
	providers := make(map[string]*ProviderConfig)
	for _, r := range ordered {
		if r.Kind == "AWSProvider" {
			providers[r.Metadata.Name] = NewProviderConfigFromResource(r)
		}
	}

	for _, r := range ordered {
		if r.Kind == "AWSProvider" {
			continue
		}

		e.output.WriteVerbose("Processando %s/%s...", r.Kind, r.Metadata.Name)

		existingState, err := e.stateManager.LoadState(r.Kind, r.Metadata.Namespace, r.Metadata.Name)
		if err != nil {
			result.Failed = append(result.Failed, ResourceResult{
				Kind:  r.Kind,
				Name:  r.Metadata.Name,
				Error: err.Error(),
			})
			continue
		}

		if existingState != nil {
			e.output.WriteVerbose("  Recurso já existe, pulando...")
			result.Skipped = append(result.Skipped, ResourceResult{
				Kind:         r.Kind,
				Name:         r.Metadata.Name,
				AWSResources: existingState.AWSResources,
				Message:      "já existe",
			})
			continue
		}

		if e.dryRun {
			result.Skipped = append(result.Skipped, ResourceResult{
				Kind:    r.Kind,
				Name:    r.Metadata.Name,
				Message: "dry-run",
			})
			continue
		}

		state, err := e.createResource(ctx, r, providers)
		if err != nil {
			result.Failed = append(result.Failed, ResourceResult{
				Kind:  r.Kind,
				Name:  r.Metadata.Name,
				Error: err.Error(),
			})
			continue
		}

		if err := e.stateManager.SaveState(state); err != nil {
			result.Failed = append(result.Failed, ResourceResult{
				Kind:  r.Kind,
				Name:  r.Metadata.Name,
				Error: fmt.Sprintf("falha ao salvar estado: %v", err),
			})
			continue
		}

		result.Created = append(result.Created, ResourceResult{
			Kind:         r.Kind,
			Name:         r.Metadata.Name,
			Namespace:    r.Metadata.Namespace,
			AWSResources: state.AWSResources,
		})
	}

	return result, nil
}

// Delete deleta os recursos
func (e *Engine) Delete(ctx context.Context, resources []Resource) (*DeleteResult, error) {
	// Ordem reversa para deleção
	ordered := OrderByDependency(resources)
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}

	result := &DeleteResult{}

	for _, r := range ordered {
		if r.Kind == "AWSProvider" {
			continue
		}

		e.output.WriteVerbose("Deletando %s/%s...", r.Kind, r.Metadata.Name)

		existingState, err := e.stateManager.LoadState(r.Kind, r.Metadata.Namespace, r.Metadata.Name)
		if err != nil {
			result.Failed = append(result.Failed, ResourceResult{
				Kind:  r.Kind,
				Name:  r.Metadata.Name,
				Error: err.Error(),
			})
			continue
		}

		if existingState == nil {
			result.Skipped = append(result.Skipped, ResourceResult{
				Kind:    r.Kind,
				Name:    r.Metadata.Name,
				Message: "não encontrado no estado",
			})
			continue
		}

		if e.dryRun {
			result.Skipped = append(result.Skipped, ResourceResult{
				Kind:    r.Kind,
				Name:    r.Metadata.Name,
				Message: "dry-run",
			})
			continue
		}

		if err := e.deleteResource(ctx, existingState); err != nil {
			result.Failed = append(result.Failed, ResourceResult{
				Kind:  r.Kind,
				Name:  r.Metadata.Name,
				Error: err.Error(),
			})
			continue
		}

		if err := e.stateManager.DeleteState(r.Kind, r.Metadata.Namespace, r.Metadata.Name); err != nil {
			result.Failed = append(result.Failed, ResourceResult{
				Kind:  r.Kind,
				Name:  r.Metadata.Name,
				Error: fmt.Sprintf("falha ao deletar estado: %v", err),
			})
			continue
		}

		result.Deleted = append(result.Deleted, ResourceResult{
			Kind:      r.Kind,
			Name:      r.Metadata.Name,
			Namespace: r.Metadata.Namespace,
		})
	}

	return result, nil
}

// Get lista recursos do estado
func (e *Engine) Get(ctx context.Context, kind string) ([]*ResourceState, error) {
	if kind == "" || kind == "all" {
		return e.stateManager.ListAllStates()
	}
	return e.stateManager.ListStates(kind)
}

// createResource cria um recurso AWS baseado no kind
func (e *Engine) createResource(ctx context.Context, r Resource, providers map[string]*ProviderConfig) (*ResourceState, error) {
	state := StateFromResource(r)

	switch r.Kind {
	case "VPC":
		return e.createVPC(ctx, r, state)
	case "Subnet":
		return e.createSubnet(ctx, r, state)
	case "InternetGateway":
		return e.createInternetGateway(ctx, r, state)
	case "SecurityGroup":
		return e.createSecurityGroup(ctx, r, state)
	case "EC2Instance":
		return e.createEC2Instance(ctx, r, state)
	case "ComputeStack":
		return e.createComputeStack(ctx, r, state)
	default:
		return nil, fmt.Errorf("tipo de recurso não suportado: %s", r.Kind)
	}
}

// deleteResource deleta um recurso AWS baseado no estado
func (e *Engine) deleteResource(ctx context.Context, state *ResourceState) error {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	switch state.Kind {
	case "VPC":
		if vpcID := state.AWSResources["vpcId"]; vpcID != "" {
			_, err := ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
				VpcId: aws.String(vpcID),
			})
			return err
		}
	case "Subnet":
		if subnetID := state.AWSResources["subnetId"]; subnetID != "" {
			_, err := ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
				SubnetId: aws.String(subnetID),
			})
			return err
		}
	case "InternetGateway":
		if igwID := state.AWSResources["internetGatewayId"]; igwID != "" {
			if vpcID := state.AWSResources["vpcId"]; vpcID != "" {
				ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
					InternetGatewayId: aws.String(igwID),
					VpcId:             aws.String(vpcID),
				})
			}
			_, err := ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
				InternetGatewayId: aws.String(igwID),
			})
			return err
		}
	case "SecurityGroup":
		if sgID := state.AWSResources["securityGroupId"]; sgID != "" {
			_, err := ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
				GroupId: aws.String(sgID),
			})
			return err
		}
	case "EC2Instance":
		if instanceID := state.AWSResources["instanceId"]; instanceID != "" {
			_, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
				InstanceIds: []string{instanceID},
			})
			return err
		}
	case "ComputeStack":
		return e.deleteComputeStack(ctx, state)
	}

	return nil
}

// createVPC cria uma VPC
func (e *Engine) createVPC(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	cidrBlock, _ := r.Spec["cidrBlock"].(string)
	if cidrBlock == "" {
		cidrBlock = "10.0.0.0/16"
	}

	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(cidrBlock),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name)},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
				},
			},
		},
	}

	result, err := ec2Client.CreateVpc(ctx, input)
	if err != nil {
		return nil, err
	}

	state.AWSResources["vpcId"] = *result.Vpc.VpcId
	state.Status = map[string]interface{}{
		"id":    *result.Vpc.VpcId,
		"state": string(result.Vpc.State),
		"ready": result.Vpc.State == types.VpcStateAvailable,
	}

	if enableDNS, ok := r.Spec["enableDnsSupport"].(bool); ok && enableDNS {
		ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
			VpcId:            result.Vpc.VpcId,
			EnableDnsSupport: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
	}
	if enableDNSHostnames, ok := r.Spec["enableDnsHostnames"].(bool); ok && enableDNSHostnames {
		ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
			VpcId:              result.Vpc.VpcId,
			EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
	}

	return state, nil
}

// createSubnet cria uma Subnet
func (e *Engine) createSubnet(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	vpcID, _ := r.Spec["vpcId"].(string)
	cidrBlock, _ := r.Spec["cidrBlock"].(string)
	az, _ := r.Spec["availabilityZone"].(string)

	input := &ec2.CreateSubnetInput{
		VpcId:            aws.String(vpcID),
		CidrBlock:        aws.String(cidrBlock),
		AvailabilityZone: aws.String(az),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSubnet,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name)},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
				},
			},
		},
	}

	result, err := ec2Client.CreateSubnet(ctx, input)
	if err != nil {
		return nil, err
	}

	state.AWSResources["subnetId"] = *result.Subnet.SubnetId
	state.AWSResources["vpcId"] = vpcID
	state.Status = map[string]interface{}{
		"id":    *result.Subnet.SubnetId,
		"state": string(result.Subnet.State),
	}

	return state, nil
}

// createInternetGateway cria um Internet Gateway
func (e *Engine) createInternetGateway(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	input := &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInternetGateway,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name)},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
				},
			},
		},
	}

	result, err := ec2Client.CreateInternetGateway(ctx, input)
	if err != nil {
		return nil, err
	}

	state.AWSResources["internetGatewayId"] = *result.InternetGateway.InternetGatewayId

	if vpcID, ok := r.Spec["vpcId"].(string); ok && vpcID != "" {
		_, err = ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
			InternetGatewayId: result.InternetGateway.InternetGatewayId,
			VpcId:             aws.String(vpcID),
		})
		if err != nil {
			return nil, fmt.Errorf("falha ao anexar IGW à VPC: %w", err)
		}
		state.AWSResources["vpcId"] = vpcID
	}

	state.Status = map[string]interface{}{
		"id": *result.InternetGateway.InternetGatewayId,
	}

	return state, nil
}

// createSecurityGroup cria um Security Group
func (e *Engine) createSecurityGroup(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	vpcID, _ := r.Spec["vpcId"].(string)
	description, _ := r.Spec["description"].(string)
	if description == "" {
		description = "Gerenciado por infra-operator"
	}

	input := &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(r.Metadata.Name),
		Description: aws.String(description),
		VpcId:       aws.String(vpcID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name)},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
				},
			},
		},
	}

	result, err := ec2Client.CreateSecurityGroup(ctx, input)
	if err != nil {
		return nil, err
	}

	state.AWSResources["securityGroupId"] = *result.GroupId
	state.AWSResources["vpcId"] = vpcID
	state.Status = map[string]interface{}{
		"id": *result.GroupId,
	}

	if ingressRules, ok := r.Spec["ingressRules"].([]interface{}); ok {
		for _, rule := range ingressRules {
			if ruleMap, ok := rule.(map[string]interface{}); ok {
				ipPermission := types.IpPermission{
					IpProtocol: aws.String(getStringFromMap(ruleMap, "protocol", "tcp")),
					FromPort:   aws.Int32(int32(getIntFromMap(ruleMap, "fromPort", 0))),
					ToPort:     aws.Int32(int32(getIntFromMap(ruleMap, "toPort", 0))),
				}
				if cidr := getStringFromMap(ruleMap, "cidrIp", ""); cidr != "" {
					ipPermission.IpRanges = []types.IpRange{{CidrIp: aws.String(cidr)}}
				}
				ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
					GroupId:       result.GroupId,
					IpPermissions: []types.IpPermission{ipPermission},
				})
			}
		}
	}

	return state, nil
}

// createEC2Instance cria uma instância EC2
func (e *Engine) createEC2Instance(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	instanceType, _ := r.Spec["instanceType"].(string)
	if instanceType == "" {
		instanceType = "t3.micro"
	}
	imageID, _ := r.Spec["imageId"].(string)
	subnetID, _ := r.Spec["subnetId"].(string)
	keyName, _ := r.Spec["keyName"].(string)

	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(imageID),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		SubnetId:     aws.String(subnetID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name)},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
				},
			},
		},
	}

	if keyName != "" {
		input.KeyName = aws.String(keyName)
	}

	if sgIDs, ok := r.Spec["securityGroupIds"].([]interface{}); ok {
		var sgs []string
		for _, sg := range sgIDs {
			if sgStr, ok := sg.(string); ok {
				sgs = append(sgs, sgStr)
			}
		}
		input.SecurityGroupIds = sgs
	}

	if userData, ok := r.Spec["userData"].(string); ok && userData != "" {
		input.UserData = aws.String(base64.StdEncoding.EncodeToString([]byte(userData)))
	}

	result, err := ec2Client.RunInstances(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(result.Instances) == 0 {
		return nil, fmt.Errorf("nenhuma instância criada")
	}

	instance := result.Instances[0]
	state.AWSResources["instanceId"] = *instance.InstanceId
	state.Status = map[string]interface{}{
		"id":    *instance.InstanceId,
		"state": string(instance.State.Name),
	}

	return state, nil
}

// createComputeStack cria um ComputeStack completo
func (e *Engine) createComputeStack(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	vpcCIDR, _ := r.Spec["vpcCIDR"].(string)
	if vpcCIDR == "" {
		vpcCIDR = "10.0.0.0/16"
	}

	e.output.Write("  Criando VPC...")
	vpcInput := &ec2.CreateVpcInput{
		CidrBlock: aws.String(vpcCIDR),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-vpc")},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
					{Key: aws.String("ComputeStack"), Value: aws.String(r.Metadata.Name)},
				},
			},
		},
	}

	vpcResult, err := ec2Client.CreateVpc(ctx, vpcInput)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar VPC: %w", err)
	}
	vpcID := *vpcResult.Vpc.VpcId
	state.AWSResources["vpcId"] = vpcID
	e.output.Write("    VPC: %s", vpcID)

	time.Sleep(2 * time.Second)

	ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:            aws.String(vpcID),
		EnableDnsSupport: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})
	ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:              aws.String(vpcID),
		EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})

	e.output.Write("  Criando Internet Gateway...")
	igwResult, err := ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInternetGateway,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-igw")},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
					{Key: aws.String("ComputeStack"), Value: aws.String(r.Metadata.Name)},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao criar IGW: %w", err)
	}
	igwID := *igwResult.InternetGateway.InternetGatewayId
	state.AWSResources["internetGatewayId"] = igwID
	e.output.Write("    IGW: %s", igwID)

	_, err = ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
		VpcId:             aws.String(vpcID),
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao anexar IGW: %w", err)
	}

	azResult, _ := ec2Client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{})
	az := "us-east-1a"
	if len(azResult.AvailabilityZones) > 0 {
		az = *azResult.AvailabilityZones[0].ZoneName
	}

	e.output.Write("  Criando Subnet Pública...")
	subnetCIDR := strings.Replace(vpcCIDR, ".0.0/16", ".1.0/24", 1)
	subnetResult, err := ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId:            aws.String(vpcID),
		CidrBlock:        aws.String(subnetCIDR),
		AvailabilityZone: aws.String(az),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSubnet,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-public-subnet")},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
					{Key: aws.String("ComputeStack"), Value: aws.String(r.Metadata.Name)},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao criar subnet: %w", err)
	}
	subnetID := *subnetResult.Subnet.SubnetId
	state.AWSResources["publicSubnetId"] = subnetID
	e.output.Write("    Subnet: %s", subnetID)

	ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		SubnetId:            aws.String(subnetID),
		MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})

	e.output.Write("  Criando Route Table...")
	rtResult, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		VpcId: aws.String(vpcID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeRouteTable,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-public-rt")},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
					{Key: aws.String("ComputeStack"), Value: aws.String(r.Metadata.Name)},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao criar route table: %w", err)
	}
	rtID := *rtResult.RouteTable.RouteTableId
	state.AWSResources["routeTableId"] = rtID
	e.output.Write("    Route Table: %s", rtID)

	ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         aws.String(rtID),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwID),
	})

	ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(rtID),
		SubnetId:     aws.String(subnetID),
	})

	bastionSpec, hasBastionSpec := r.Spec["bastionInstance"].(map[string]interface{})
	bastionEnabled := false
	if hasBastionSpec {
		if enabled, ok := bastionSpec["enabled"].(bool); ok {
			bastionEnabled = enabled
		}
	}

	if bastionEnabled {
		e.output.Write("  Criando Security Group...")
		sgResult, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(r.Metadata.Name + "-bastion-sg"),
			Description: aws.String("Security group para bastion host"),
			VpcId:       aws.String(vpcID),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSecurityGroup,
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-bastion-sg")},
						{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
						{Key: aws.String("ComputeStack"), Value: aws.String(r.Metadata.Name)},
					},
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("falha ao criar security group: %w", err)
		}
		sgID := *sgResult.GroupId
		state.AWSResources["bastionSecurityGroupId"] = sgID
		e.output.Write("    Security Group: %s", sgID)

		ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(sgID),
			IpPermissions: []types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(22),
					ToPort:     aws.Int32(22),
					IpRanges:   []types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
				},
			},
		})

		imageID := ""
		if imgID, ok := bastionSpec["imageId"].(string); ok && imgID != "" {
			imageID = imgID
		} else {
			amiResult, _ := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
				Owners: []string{"amazon"},
				Filters: []types.Filter{
					{Name: aws.String("name"), Values: []string{"amzn2-ami-hvm-*-x86_64-gp2"}},
					{Name: aws.String("state"), Values: []string{"available"}},
				},
			})
			if len(amiResult.Images) > 0 {
				imageID = *amiResult.Images[0].ImageId
			}
		}

		if imageID == "" {
			e.output.Write("  Aviso: Nenhuma AMI encontrada, pulando instância bastion")
		} else {
			instanceType := "t3.micro"
			if iType, ok := bastionSpec["instanceType"].(string); ok && iType != "" {
				instanceType = iType
			}

			e.output.Write("  Criando Instância Bastion...")
			instanceInput := &ec2.RunInstancesInput{
				ImageId:          aws.String(imageID),
				InstanceType:     types.InstanceType(instanceType),
				MinCount:         aws.Int32(1),
				MaxCount:         aws.Int32(1),
				SubnetId:         aws.String(subnetID),
				SecurityGroupIds: []string{sgID},
				TagSpecifications: []types.TagSpecification{
					{
						ResourceType: types.ResourceTypeInstance,
						Tags: []types.Tag{
							{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-bastion")},
							{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator")},
							{Key: aws.String("ComputeStack"), Value: aws.String(r.Metadata.Name)},
						},
					},
				},
			}

			if keyName, ok := bastionSpec["keyName"].(string); ok && keyName != "" {
				instanceInput.KeyName = aws.String(keyName)
			}

			if userData, ok := bastionSpec["userData"].(string); ok && userData != "" {
				instanceInput.UserData = aws.String(base64.StdEncoding.EncodeToString([]byte(userData)))
			}

			instanceResult, err := ec2Client.RunInstances(ctx, instanceInput)
			if err != nil {
				return nil, fmt.Errorf("falha ao criar instância bastion: %w", err)
			}

			if len(instanceResult.Instances) > 0 {
				instanceID := *instanceResult.Instances[0].InstanceId
				state.AWSResources["bastionInstanceId"] = instanceID
				e.output.Write("    Instance: %s", instanceID)
			}
		}
	}

	state.Status = map[string]interface{}{
		"phase":   "Ready",
		"ready":   true,
		"message": "ComputeStack criado com sucesso",
		"vpc": map[string]interface{}{
			"id":    vpcID,
			"state": "available",
		},
	}

	return state, nil
}

// deleteComputeStack deleta todos os recursos do ComputeStack
func (e *Engine) deleteComputeStack(ctx context.Context, state *ResourceState) error {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	if instanceID := state.AWSResources["bastionInstanceId"]; instanceID != "" {
		e.output.Write("  Terminando instância bastion %s...", instanceID)
		ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{instanceID},
		})
		time.Sleep(10 * time.Second)
	}

	if sgID := state.AWSResources["bastionSecurityGroupId"]; sgID != "" {
		e.output.Write("  Deletando security group %s...", sgID)
		ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sgID),
		})
	}

	if rtID := state.AWSResources["routeTableId"]; rtID != "" {
		e.output.Write("  Deletando route table %s...", rtID)
		rtResult, _ := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			RouteTableIds: []string{rtID},
		})
		if len(rtResult.RouteTables) > 0 {
			for _, assoc := range rtResult.RouteTables[0].Associations {
				if !*assoc.Main {
					ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
						AssociationId: assoc.RouteTableAssociationId,
					})
				}
			}
		}
		ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
			RouteTableId: aws.String(rtID),
		})
	}

	if subnetID := state.AWSResources["publicSubnetId"]; subnetID != "" {
		e.output.Write("  Deletando subnet %s...", subnetID)
		ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
			SubnetId: aws.String(subnetID),
		})
	}

	if igwID := state.AWSResources["internetGatewayId"]; igwID != "" {
		if vpcID := state.AWSResources["vpcId"]; vpcID != "" {
			e.output.Write("  Desanexando IGW %s...", igwID)
			ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
				InternetGatewayId: aws.String(igwID),
				VpcId:             aws.String(vpcID),
			})
		}
		e.output.Write("  Deletando IGW %s...", igwID)
		ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(igwID),
		})
	}

	if vpcID := state.AWSResources["vpcId"]; vpcID != "" {
		e.output.Write("  Deletando VPC %s...", vpcID)
		ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
			VpcId: aws.String(vpcID),
		})
	}

	return nil
}

func getStringFromMap(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}

func getIntFromMap(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}
