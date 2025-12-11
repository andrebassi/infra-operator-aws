package cli

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

// Executor gerencia a criação/deleção de recursos no modo CLI
type Executor struct {
	stateManager   *StateManager
	providerConfig *ProviderConfig
	awsConfig      aws.Config
	dryRun         bool
	verbose        bool
}

// NewExecutor cria um novo executor
func NewExecutor(stateDir string, provider *ProviderConfig, dryRun, verbose bool) (*Executor, error) {
	ctx := context.Background()
	awsCfg, err := provider.GetAWSConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter configuração AWS: %w", err)
	}

	return &Executor{
		stateManager:   NewStateManager(stateDir),
		providerConfig: provider,
		awsConfig:      awsCfg,
		dryRun:         dryRun,
		verbose:        verbose,
	}, nil
}

// Apply cria ou atualiza recursos de um arquivo de manifesto
func (e *Executor) Apply(ctx context.Context, resources []Resource) error {
	// Ordena por dependência
	ordered := OrderByDependency(resources)

	// Extrai configurações de provider primeiro
	providers := make(map[string]*ProviderConfig)
	for _, r := range ordered {
		if r.Kind == "AWSProvider" {
			providers[r.Metadata.Name] = NewProviderConfigFromResource(r)
		}
	}

	// Processa cada recurso
	for _, r := range ordered {
		if r.Kind == "AWSProvider" {
			continue // Pula recursos de provider, são apenas configuração
		}

		e.log("Processando %s/%s...", r.Kind, r.Metadata.Name)

		// Verifica se o recurso já existe no estado
		existingState, err := e.stateManager.LoadState(r.Kind, r.Metadata.Namespace, r.Metadata.Name)
		if err != nil {
			return fmt.Errorf("falha ao carregar estado para %s/%s: %w", r.Kind, r.Metadata.Name, err)
		}

		if existingState != nil {
			e.log("  Recurso já existe no estado, verificando mudanças...")
			// TODO: Implementar lógica de atualização
			continue
		}

		// Cria o recurso
		if e.dryRun {
			fmt.Printf("  [DRY-RUN] Criaria %s/%s\n", r.Kind, r.Metadata.Name)
			continue
		}

		state, err := e.createResource(ctx, r, providers)
		if err != nil {
			return fmt.Errorf("falha ao criar %s/%s: %w", r.Kind, r.Metadata.Name, err)
		}

		// Salva estado
		if err := e.stateManager.SaveState(state); err != nil {
			return fmt.Errorf("falha ao salvar estado para %s/%s: %w", r.Kind, r.Metadata.Name, err)
		}

		fmt.Printf("  Criado %s/%s\n", r.Kind, r.Metadata.Name)
		if state.AWSResources != nil {
			for k, v := range state.AWSResources {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
	}

	return nil
}

// Plan mostra o que seria criado/atualizado/deletado
func (e *Executor) Plan(ctx context.Context, resources []Resource) error {
	ordered := OrderByDependency(resources)

	fmt.Println("\n=== Plano de Execução ===\n")

	var toCreate, toUpdate, noChange int

	for _, r := range ordered {
		if r.Kind == "AWSProvider" {
			continue
		}

		existingState, _ := e.stateManager.LoadState(r.Kind, r.Metadata.Namespace, r.Metadata.Name)

		if existingState == nil {
			fmt.Printf("  + %s/%s (CRIAR)\n", r.Kind, r.Metadata.Name)
			toCreate++
		} else {
			// TODO: Comparar specs para mudanças
			fmt.Printf("  ~ %s/%s (SEM MUDANÇA)\n", r.Kind, r.Metadata.Name)
			noChange++
		}
	}

	fmt.Printf("\nPlano: %d para criar, %d para atualizar, %d sem mudança\n", toCreate, toUpdate, noChange)
	return nil
}

// Delete deleta recursos de um arquivo de manifesto
func (e *Executor) Delete(ctx context.Context, resources []Resource) error {
	// Ordem reversa para deleção
	ordered := OrderByDependency(resources)
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}

	for _, r := range ordered {
		if r.Kind == "AWSProvider" {
			continue
		}

		e.log("Deletando %s/%s...", r.Kind, r.Metadata.Name)

		existingState, err := e.stateManager.LoadState(r.Kind, r.Metadata.Namespace, r.Metadata.Name)
		if err != nil {
			return fmt.Errorf("falha ao carregar estado para %s/%s: %w", r.Kind, r.Metadata.Name, err)
		}

		if existingState == nil {
			e.log("  Recurso não está no estado, pulando")
			continue
		}

		if e.dryRun {
			fmt.Printf("  [DRY-RUN] Deletaria %s/%s\n", r.Kind, r.Metadata.Name)
			continue
		}

		if err := e.deleteResource(ctx, existingState); err != nil {
			return fmt.Errorf("falha ao deletar %s/%s: %w", r.Kind, r.Metadata.Name, err)
		}

		// Remove do estado
		if err := e.stateManager.DeleteState(r.Kind, r.Metadata.Namespace, r.Metadata.Name); err != nil {
			return fmt.Errorf("falha ao deletar estado para %s/%s: %w", r.Kind, r.Metadata.Name, err)
		}

		fmt.Printf("  Deletado %s/%s\n", r.Kind, r.Metadata.Name)
	}

	return nil
}

// createResource cria um recurso AWS baseado no kind
func (e *Executor) createResource(ctx context.Context, r Resource, providers map[string]*ProviderConfig) (*ResourceState, error) {
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
		return nil, fmt.Errorf("tipo de recurso não suportado: %s (modo CLI suporta: VPC, Subnet, InternetGateway, SecurityGroup, EC2Instance, ComputeStack)", r.Kind)
	}
}

// deleteResource deleta um recurso AWS baseado no estado
func (e *Executor) deleteResource(ctx context.Context, state *ResourceState) error {
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
			// Desanexa primeiro
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
func (e *Executor) createVPC(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
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
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
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

	// Habilita suporte DNS se solicitado
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
func (e *Executor) createSubnet(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
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
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
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
func (e *Executor) createInternetGateway(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	input := &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInternetGateway,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name)},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
				},
			},
		},
	}

	result, err := ec2Client.CreateInternetGateway(ctx, input)
	if err != nil {
		return nil, err
	}

	state.AWSResources["internetGatewayId"] = *result.InternetGateway.InternetGatewayId

	// Anexa à VPC se especificado
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
func (e *Executor) createSecurityGroup(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	vpcID, _ := r.Spec["vpcId"].(string)
	description, _ := r.Spec["description"].(string)
	if description == "" {
		description = "Gerenciado por infra-operator-cli"
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
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
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

	// Adiciona regras de ingress se especificadas
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
func (e *Executor) createEC2Instance(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
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
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
				},
			},
		},
	}

	if keyName != "" {
		input.KeyName = aws.String(keyName)
	}

	// Security groups
	if sgIDs, ok := r.Spec["securityGroupIds"].([]interface{}); ok {
		var sgs []string
		for _, sg := range sgIDs {
			if sgStr, ok := sg.(string); ok {
				sgs = append(sgs, sgStr)
			}
		}
		input.SecurityGroupIds = sgs
	}

	// User data
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
func (e *Executor) createComputeStack(ctx context.Context, r Resource, state *ResourceState) (*ResourceState, error) {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	// Obtém CIDR da VPC do spec
	vpcCIDR, _ := r.Spec["vpcCIDR"].(string)
	if vpcCIDR == "" {
		vpcCIDR = "10.0.0.0/16"
	}

	fmt.Println("  Criando VPC...")
	vpcInput := &ec2.CreateVpcInput{
		CidrBlock: aws.String(vpcCIDR),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-vpc")},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
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
	fmt.Printf("    VPC: %s\n", vpcID)

	// Aguarda VPC ficar disponível
	time.Sleep(2 * time.Second)

	// Habilita DNS
	ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:            aws.String(vpcID),
		EnableDnsSupport: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})
	ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:              aws.String(vpcID),
		EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})

	// Cria Internet Gateway
	fmt.Println("  Criando Internet Gateway...")
	igwResult, err := ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInternetGateway,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-igw")},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
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
	fmt.Printf("    IGW: %s\n", igwID)

	// Anexa IGW à VPC
	_, err = ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
		VpcId:             aws.String(vpcID),
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao anexar IGW: %w", err)
	}

	// Obtém AZs
	azResult, _ := ec2Client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{})
	az := "us-east-1a"
	if len(azResult.AvailabilityZones) > 0 {
		az = *azResult.AvailabilityZones[0].ZoneName
	}

	// Cria subnet pública
	fmt.Println("  Criando Subnet Pública...")
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
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
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
	fmt.Printf("    Subnet: %s\n", subnetID)

	// Habilita IP público automático
	ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		SubnetId:            aws.String(subnetID),
		MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
	})

	// Cria Route Table
	fmt.Println("  Criando Route Table...")
	rtResult, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		VpcId: aws.String(vpcID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeRouteTable,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-public-rt")},
					{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
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
	fmt.Printf("    Route Table: %s\n", rtID)

	// Adiciona rota para IGW
	ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         aws.String(rtID),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwID),
	})

	// Associa route table com subnet
	ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(rtID),
		SubnetId:     aws.String(subnetID),
	})

	// Verifica se instância bastion está habilitada
	bastionSpec, hasBastionSpec := r.Spec["bastionInstance"].(map[string]interface{})
	bastionEnabled := false
	if hasBastionSpec {
		if enabled, ok := bastionSpec["enabled"].(bool); ok {
			bastionEnabled = enabled
		}
	}

	if bastionEnabled {
		// Cria Security Group para bastion
		fmt.Println("  Criando Security Group...")
		sgResult, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(r.Metadata.Name + "-bastion-sg"),
			Description: aws.String("Security group para bastion host"),
			VpcId:       aws.String(vpcID),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSecurityGroup,
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(r.Metadata.Name + "-bastion-sg")},
						{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
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
		fmt.Printf("    Security Group: %s\n", sgID)

		// Adiciona regra SSH
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

		// Obtém AMI
		imageID := ""
		if imgID, ok := bastionSpec["imageId"].(string); ok && imgID != "" {
			imageID = imgID
		} else {
			// Busca AMI Amazon Linux 2
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
			fmt.Println("  Aviso: Nenhuma AMI encontrada, pulando instância bastion")
		} else {
			// Obtém tipo de instância
			instanceType := "t3.micro"
			if iType, ok := bastionSpec["instanceType"].(string); ok && iType != "" {
				instanceType = iType
			}

			// Cria instância EC2
			fmt.Println("  Criando Instância Bastion...")
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
							{Key: aws.String("ManagedBy"), Value: aws.String("infra-operator-cli")},
							{Key: aws.String("ComputeStack"), Value: aws.String(r.Metadata.Name)},
						},
					},
				},
			}

			// Adiciona key name se especificado
			if keyName, ok := bastionSpec["keyName"].(string); ok && keyName != "" {
				instanceInput.KeyName = aws.String(keyName)
			}

			// Adiciona user data se especificado
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
				fmt.Printf("    Instance: %s\n", instanceID)
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
func (e *Executor) deleteComputeStack(ctx context.Context, state *ResourceState) error {
	ec2Client := ec2.NewFromConfig(e.awsConfig)

	// Deleta em ordem reversa da criação

	// 1. Termina instância bastion
	if instanceID := state.AWSResources["bastionInstanceId"]; instanceID != "" {
		fmt.Printf("  Terminando instância bastion %s...\n", instanceID)
		ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{instanceID},
		})
		// Aguarda terminação
		time.Sleep(10 * time.Second)
	}

	// 2. Deleta security group
	if sgID := state.AWSResources["bastionSecurityGroupId"]; sgID != "" {
		fmt.Printf("  Deletando security group %s...\n", sgID)
		ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sgID),
		})
	}

	// 3. Deleta route table (desassocia primeiro)
	if rtID := state.AWSResources["routeTableId"]; rtID != "" {
		fmt.Printf("  Deletando route table %s...\n", rtID)
		// Obtém associações
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

	// 4. Deleta subnet
	if subnetID := state.AWSResources["publicSubnetId"]; subnetID != "" {
		fmt.Printf("  Deletando subnet %s...\n", subnetID)
		ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
			SubnetId: aws.String(subnetID),
		})
	}

	// 5. Desanexa e deleta IGW
	if igwID := state.AWSResources["internetGatewayId"]; igwID != "" {
		if vpcID := state.AWSResources["vpcId"]; vpcID != "" {
			fmt.Printf("  Desanexando IGW %s...\n", igwID)
			ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
				InternetGatewayId: aws.String(igwID),
				VpcId:             aws.String(vpcID),
			})
		}
		fmt.Printf("  Deletando IGW %s...\n", igwID)
		ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(igwID),
		})
	}

	// 6. Deleta VPC
	if vpcID := state.AWSResources["vpcId"]; vpcID != "" {
		fmt.Printf("  Deletando VPC %s...\n", vpcID)
		ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
			VpcId: aws.String(vpcID),
		})
	}

	return nil
}

func (e *Executor) log(format string, args ...interface{}) {
	if e.verbose {
		fmt.Printf(format+"\n", args...)
	}
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
