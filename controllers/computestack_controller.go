package controllers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	awsclients "infra-operator/pkg/clients"
)

const computeStackFinalizerName = "computestack.aws-infra-operator.runner.codes/finalizer"

// Fases do ComputeStack
const (
	PhasePending              = "Pending"
	PhaseCreatingVPC          = "CreatingVPC"
	PhaseWaitingVPC           = "WaitingVPC"
	PhaseCreatingIGW          = "CreatingInternetGateway"
	PhaseCreatingSubnets      = "CreatingSubnets"
	PhaseWaitingSubnets       = "WaitingSubnets"
	PhaseCreatingNATGateway   = "CreatingNATGateway"
	PhaseWaitingNATGateway    = "WaitingNATGateway"
	PhaseCreatingRouteTables  = "CreatingRouteTables"
	PhaseCreatingSecGroups    = "CreatingSecurityGroups"
	PhaseCreatingBastionSG    = "CreatingBastionSecurityGroup"
	PhaseCreatingBastion      = "CreatingBastionInstance"
	PhaseWaitingBastion       = "WaitingBastionInstance"
	PhaseReady                = "Ready"
	PhaseDeleting             = "Deleting"
	PhaseFailed               = "Failed"
)

// ComputeStackReconciler reconcilia um recurso ComputeStack
type ComputeStackReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *awsclients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=computestacks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=computestacks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=computestacks/finalizers,verbs=update

func (r *ComputeStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Buscar o ComputeStack
	stack := &infrav1alpha1.ComputeStack{}
	if err := r.Get(ctx, req.NamespacedName, stack); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get EC2 client
	awsConfig, _, err := r.AWSClientFactory.GetAWSConfigFromProviderRef(ctx, stack.Namespace, stack.Spec.ProviderRef)
	if err != nil {
		logger.Error(err, "Failed to get AWS configuration")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}
	ec2Client := ec2.NewFromConfig(awsConfig)

	// Check if being deleted
	if !stack.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(stack, computeStackFinalizerName) {
			// Set phase to Deleting immediately (non-blocking for user)
			if stack.Status.Phase != PhaseDeleting {
				stack.Status.Phase = PhaseDeleting
				stack.Status.Message = "Starting deletion of AWS resources..."
				stack.Status.Ready = false
				if err := r.Status().Update(ctx, stack); err != nil {
					return ctrl.Result{}, err
				}
				logger.Info("ComputeStack deletion started", "name", stack.Name)
				return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
			}

			// Execute cleanup asynchronously (one step at a time)
			done, err := r.deleteStackAsync(ctx, ec2Client, stack)
			if err != nil {
				logger.Error(err, "Failed to delete ComputeStack resources (will retry)")
				stack.Status.Message = fmt.Sprintf("Deletion in progress: %s (retrying...)", err.Error())
				if updateErr := r.Status().Update(ctx, stack); updateErr != nil {
					logger.Error(updateErr, "Failed to update status")
				}
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil // Continue trying
			}

			if !done {
				// Still deleting resources, requeue
				if err := r.Status().Update(ctx, stack); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}

			// All resources deleted, remove finalizer
			logger.Info("All ComputeStack resources deleted, removing finalizer", "name", stack.Name)
			controllerutil.RemoveFinalizer(stack, computeStackFinalizerName)
			if err := r.Update(ctx, stack); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Adicionar finalizer
	if !controllerutil.ContainsFinalizer(stack, computeStackFinalizerName) {
		controllerutil.AddFinalizer(stack, computeStackFinalizerName)
		if err := r.Update(ctx, stack); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Inicializar fase se vazia
	if stack.Status.Phase == "" {
		stack.Status.Phase = PhasePending
		stack.Status.Message = "Initializing ComputeStack creation"
		if err := r.Status().Update(ctx, stack); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Se já está Ready, apenas verificar drift periodicamente
	if stack.Status.Phase == PhaseReady {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Processar próxima fase
	result, err := r.processNextPhase(ctx, ec2Client, stack)
	if err != nil {
		logger.Error(err, "Falha ao processar fase", "phase", stack.Status.Phase)
		stack.Status.Phase = PhaseFailed
		stack.Status.Message = err.Error()
		stack.Status.Ready = false
		if updateErr := r.Status().Update(ctx, stack); updateErr != nil {
			logger.Error(updateErr, "Falha ao atualizar status")
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	// Atualizar status
	stack.Status.LastSyncTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, stack); err != nil {
		logger.Error(err, "Falha ao atualizar status do ComputeStack")
		return ctrl.Result{}, err
	}

	logger.Info("Fase processada",
		"name", stack.Name,
		"phase", stack.Status.Phase,
		"ready", stack.Status.Ready)

	return result, nil
}

// determineCurrentPhase determina a fase correta baseado nos recursos já criados
// Isso garante idempotência mesmo com conflitos de atualização de status
func (r *ComputeStackReconciler) determineCurrentPhase(stack *infrav1alpha1.ComputeStack) string {
	// Se já está Ready ou Failed, manter
	if stack.Status.Phase == PhaseReady || stack.Status.Phase == PhaseFailed {
		return stack.Status.Phase
	}

	// Verificar estado real dos recursos criados
	hasVPC := stack.Status.VPC != nil && stack.Status.VPC.ID != ""
	vpcAvailable := hasVPC && stack.Status.VPC.State == "available"
	hasIGW := stack.Status.InternetGateway != nil && stack.Status.InternetGateway.ID != ""
	hasSubnets := len(stack.Status.PublicSubnets) > 0 || len(stack.Status.PrivateSubnets) > 0
	subnetsAvailable := hasSubnets && r.allSubnetsAvailable(stack)
	hasNATGateways := len(stack.Status.NATGateways) > 0
	hasRouteTables := len(stack.Status.RouteTables) > 0
	hasSecurityGroups := len(stack.Status.SecurityGroups) > 0
	hasBastionSG := stack.Status.BastionSecurityGroup != nil && stack.Status.BastionSecurityGroup.ID != ""
	hasBastionInstance := stack.Status.BastionInstance != nil && stack.Status.BastionInstance.ID != ""
	bastionRunning := hasBastionInstance && stack.Status.BastionInstance.State == "running"

	// Precisa de NAT Gateway?
	needsNAT := stack.Spec.NATGateway != nil && stack.Spec.NATGateway.Enabled && len(stack.Spec.PrivateSubnets) > 0
	// Precisa de Security Groups?
	needsSecGroups := len(stack.Spec.DefaultSecurityGroups) > 0
	// Precisa de Bastion Instance?
	needsBastion := stack.Spec.BastionInstance != nil && stack.Spec.BastionInstance.Enabled

	// Determinar fase baseado no que já foi criado e seus estados
	if !hasVPC {
		return PhaseCreatingVPC
	}
	if !vpcAvailable {
		return PhaseWaitingVPC
	}
	if !hasIGW {
		return PhaseCreatingIGW
	}
	if !hasSubnets {
		return PhaseCreatingSubnets
	}
	if !subnetsAvailable {
		return PhaseWaitingSubnets
	}
	if needsNAT && !hasNATGateways {
		return PhaseCreatingNATGateway
	}
	// Verificar se NAT está disponível
	if needsNAT && hasNATGateways {
		for _, nat := range stack.Status.NATGateways {
			if nat.State != "available" {
				return PhaseWaitingNATGateway
			}
		}
	}
	if !hasRouteTables {
		return PhaseCreatingRouteTables
	}
	if needsSecGroups && !hasSecurityGroups {
		return PhaseCreatingSecGroups
	}
	// Verificar fases de Bastion
	if needsBastion && !hasBastionSG {
		return PhaseCreatingBastionSG
	}
	if needsBastion && !hasBastionInstance {
		return PhaseCreatingBastion
	}
	if needsBastion && !bastionRunning {
		return PhaseWaitingBastion
	}

	// Tudo criado = Ready
	return PhaseReady
}

// allSubnetsAvailable verifica se todas as subnets estão available
func (r *ComputeStackReconciler) allSubnetsAvailable(stack *infrav1alpha1.ComputeStack) bool {
	for _, subnet := range stack.Status.PublicSubnets {
		if subnet.State != "available" {
			return false
		}
	}
	for _, subnet := range stack.Status.PrivateSubnets {
		if subnet.State != "available" {
			return false
		}
	}
	return true
}

// checkVPCReady verifica se a VPC está no estado "available"
func (r *ComputeStackReconciler) checkVPCReady(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) (bool, error) {
	if stack.Status.VPC == nil || stack.Status.VPC.ID == "" {
		return false, nil
	}

	output, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{stack.Status.VPC.ID},
	})
	if err != nil {
		return false, err
	}

	if len(output.Vpcs) == 0 {
		return false, fmt.Errorf("VPC %s not found", stack.Status.VPC.ID)
	}

	vpc := output.Vpcs[0]
	stack.Status.VPC.State = string(vpc.State)

	return vpc.State == types.VpcStateAvailable, nil
}

// checkSubnetsReady verifica se todas as subnets estão no estado "available"
func (r *ComputeStackReconciler) checkSubnetsReady(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) (bool, error) {
	// Coletar todos os IDs de subnets
	var subnetIDs []string
	for _, subnet := range stack.Status.PublicSubnets {
		if subnet.ID != "" {
			subnetIDs = append(subnetIDs, subnet.ID)
		}
	}
	for _, subnet := range stack.Status.PrivateSubnets {
		if subnet.ID != "" {
			subnetIDs = append(subnetIDs, subnet.ID)
		}
	}

	if len(subnetIDs) == 0 {
		return false, nil
	}

	output, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: subnetIDs,
	})
	if err != nil {
		return false, err
	}

	// Criar mapa para lookup rápido
	subnetStates := make(map[string]types.SubnetState)
	for _, subnet := range output.Subnets {
		subnetStates[aws.ToString(subnet.SubnetId)] = subnet.State
	}

	// Atualizar estados das subnets públicas
	allReady := true
	for i := range stack.Status.PublicSubnets {
		if state, ok := subnetStates[stack.Status.PublicSubnets[i].ID]; ok {
			stack.Status.PublicSubnets[i].State = string(state)
			if state != types.SubnetStateAvailable {
				allReady = false
			}
		}
	}

	// Atualizar estados das subnets privadas
	for i := range stack.Status.PrivateSubnets {
		if state, ok := subnetStates[stack.Status.PrivateSubnets[i].ID]; ok {
			stack.Status.PrivateSubnets[i].State = string(state)
			if state != types.SubnetStateAvailable {
				allReady = false
			}
		}
	}

	return allReady, nil
}

// processNextPhase processa a próxima fase baseado no estado atual
func (r *ComputeStackReconciler) processNextPhase(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Determinar fase correta baseado nos recursos existentes (idempotência)
	correctPhase := r.determineCurrentPhase(stack)
	if correctPhase != stack.Status.Phase {
		logger.Info("Corrigindo fase baseado no estado real dos recursos",
			"savedPhase", stack.Status.Phase,
			"correctPhase", correctPhase)
		stack.Status.Phase = correctPhase
	}

	switch stack.Status.Phase {
	case PhasePending:
		// Próxima fase: criar VPC
		stack.Status.Phase = PhaseCreatingVPC
		stack.Status.Message = "Creating VPC..."
		return ctrl.Result{Requeue: true}, nil

	case PhaseCreatingVPC:
		logger.Info("Step 1/7: Creating VPC")
		if err := r.reconcileVPC(ctx, ec2Client, stack); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create VPC: %w", err)
		}
		stack.Status.Phase = PhaseWaitingVPC
		stack.Status.Message = "VPC created. Waiting for VPC to become available..."
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil

	case PhaseWaitingVPC:
		logger.Info("Step 1/7: Waiting for VPC to become available")
		ready, err := r.checkVPCReady(ctx, ec2Client, stack)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check VPC status: %w", err)
		}
		if !ready {
			stack.Status.Message = fmt.Sprintf("Waiting for VPC %s to become available (state: %s)...", stack.Status.VPC.ID, stack.Status.VPC.State)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		stack.Status.Phase = PhaseCreatingIGW
		stack.Status.Message = "VPC available. Creating Internet Gateway..."
		return ctrl.Result{Requeue: true}, nil

	case PhaseCreatingIGW:
		logger.Info("Step 2/7: Creating Internet Gateway")
		if err := r.reconcileInternetGateway(ctx, ec2Client, stack); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Internet Gateway: %w", err)
		}
		stack.Status.Phase = PhaseCreatingSubnets
		stack.Status.Message = "Internet Gateway created. Creating Subnets..."
		return ctrl.Result{Requeue: true}, nil

	case PhaseCreatingSubnets:
		logger.Info("Step 3/7: Creating Subnets")
		if err := r.reconcileSubnets(ctx, ec2Client, stack); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Subnets: %w", err)
		}
		stack.Status.Phase = PhaseWaitingSubnets
		stack.Status.Message = "Subnets created. Waiting for subnets to become available..."
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil

	case PhaseWaitingSubnets:
		logger.Info("Step 3/7: Waiting for Subnets to become available")
		ready, err := r.checkSubnetsReady(ctx, ec2Client, stack)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check Subnets status: %w", err)
		}
		if !ready {
			stack.Status.Message = "Waiting for subnets to become available..."
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		// Verificar se precisa criar NAT Gateway
		if stack.Spec.NATGateway != nil && stack.Spec.NATGateway.Enabled && len(stack.Spec.PrivateSubnets) > 0 {
			stack.Status.Phase = PhaseCreatingNATGateway
			stack.Status.Message = "Subnets available. Creating NAT Gateway..."
		} else {
			stack.Status.Phase = PhaseCreatingRouteTables
			stack.Status.Message = "Subnets available. Creating Route Tables..."
		}
		return ctrl.Result{Requeue: true}, nil

	case PhaseCreatingNATGateway:
		logger.Info("Step 4/7: Creating NAT Gateway")
		if err := r.createNATGateways(ctx, ec2Client, stack); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create NAT Gateway: %w", err)
		}
		stack.Status.Phase = PhaseWaitingNATGateway
		stack.Status.Message = "NAT Gateway created. Waiting for it to become available..."
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil

	case PhaseWaitingNATGateway:
		logger.Info("Step 4/7: Waiting for NAT Gateway to become available")
		ready, err := r.checkNATGatewaysReady(ctx, ec2Client, stack)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check NAT Gateway status: %w", err)
		}
		if !ready {
			stack.Status.Message = "Waiting for NAT Gateway to become available..."
			return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}
		stack.Status.Phase = PhaseCreatingRouteTables
		stack.Status.Message = "NAT Gateway available. Creating Route Tables..."
		return ctrl.Result{Requeue: true}, nil

	case PhaseCreatingRouteTables:
		logger.Info("Step 5/9: Creating Route Tables")
		if err := r.reconcileRouteTables(ctx, ec2Client, stack); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Route Tables: %w", err)
		}
		// Verificar se precisa criar Security Groups
		if len(stack.Spec.DefaultSecurityGroups) > 0 {
			stack.Status.Phase = PhaseCreatingSecGroups
			stack.Status.Message = "Route Tables created. Creating Security Groups..."
		} else if stack.Spec.BastionInstance != nil && stack.Spec.BastionInstance.Enabled {
			// Pular security groups gerais mas criar bastion
			stack.Status.Phase = PhaseCreatingBastionSG
			stack.Status.Message = "Route Tables created. Creating Bastion Security Group..."
		} else {
			stack.Status.Phase = PhaseReady
			stack.Status.Ready = true
			stack.Status.Message = "ComputeStack created successfully"
		}
		return ctrl.Result{Requeue: true}, nil

	case PhaseCreatingSecGroups:
		logger.Info("Step 6/9: Creating Security Groups")
		if err := r.reconcileSecurityGroups(ctx, ec2Client, stack); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Security Groups: %w", err)
		}
		// Verificar se precisa criar Bastion
		if stack.Spec.BastionInstance != nil && stack.Spec.BastionInstance.Enabled {
			stack.Status.Phase = PhaseCreatingBastionSG
			stack.Status.Message = "Security Groups created. Creating Bastion Security Group..."
		} else {
			stack.Status.Phase = PhaseReady
			stack.Status.Ready = true
			stack.Status.Message = "ComputeStack created successfully"
		}
		return ctrl.Result{Requeue: true}, nil

	case PhaseCreatingBastionSG:
		logger.Info("Step 7/9: Creating Bastion Security Group")
		if err := r.reconcileBastionSecurityGroup(ctx, ec2Client, stack); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Bastion Security Group: %w", err)
		}
		stack.Status.Phase = PhaseCreatingBastion
		stack.Status.Message = "Bastion Security Group created. Creating Bastion Instance..."
		return ctrl.Result{Requeue: true}, nil

	case PhaseCreatingBastion:
		logger.Info("Step 8/9: Creating Bastion Instance")
		if err := r.reconcileBastionInstance(ctx, ec2Client, stack); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Bastion Instance: %w", err)
		}
		stack.Status.Phase = PhaseWaitingBastion
		stack.Status.Message = "Bastion Instance created. Waiting for instance to become running..."
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil

	case PhaseWaitingBastion:
		logger.Info("Step 9/9: Waiting for Bastion Instance to become running")
		ready, err := r.checkBastionReady(ctx, ec2Client, stack)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check Bastion Instance status: %w", err)
		}
		if !ready {
			stack.Status.Message = fmt.Sprintf("Waiting for Bastion Instance %s to become running (state: %s)...",
				stack.Status.BastionInstance.ID, stack.Status.BastionInstance.State)
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		stack.Status.Phase = PhaseReady
		stack.Status.Ready = true
		stack.Status.Message = fmt.Sprintf("ComputeStack created successfully. SSH: %s", stack.Status.BastionInstance.SSHCommand)
		return ctrl.Result{}, nil

	case PhaseFailed:
		// Tentar novamente após um tempo
		stack.Status.Phase = PhasePending
		stack.Status.Message = "Retrying creation..."
		return ctrl.Result{Requeue: true}, nil

	default:
		// Fase desconhecida, reiniciar
		stack.Status.Phase = PhasePending
		return ctrl.Result{Requeue: true}, nil
	}
}

// reconcileVPC cria ou atualiza a VPC
func (r *ComputeStackReconciler) reconcileVPC(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) error {
	logger := log.FromContext(ctx)

	// Verificar se VPC já existe no status
	if stack.Status.VPC != nil && stack.Status.VPC.ID != "" {
		// Verificar se ainda existe na AWS
		output, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
			VpcIds: []string{stack.Status.VPC.ID},
		})
		if err == nil && len(output.Vpcs) > 0 {
			stack.Status.VPC.State = string(output.Vpcs[0].State)
			return nil
		}
	}

	// Se foi especificado um VPC ID existente, usar ao invés de criar
	if stack.Spec.ExistingVpcID != "" {
		logger.Info("Usando VPC existente", "vpcID", stack.Spec.ExistingVpcID)

		// Verificar se a VPC existe na AWS
		output, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
			VpcIds: []string{stack.Spec.ExistingVpcID},
		})
		if err != nil {
			return fmt.Errorf("existing VPC %s not found: %w", stack.Spec.ExistingVpcID, err)
		}
		if len(output.Vpcs) == 0 {
			return fmt.Errorf("existing VPC %s not found", stack.Spec.ExistingVpcID)
		}

		vpc := output.Vpcs[0]
		stack.Status.VPC = &infrav1alpha1.VPCStatusInfo{
			ID:    stack.Spec.ExistingVpcID,
			CIDR:  aws.ToString(vpc.CidrBlock),
			State: string(vpc.State),
		}
		return nil
	}

	// Validar que temos VpcCIDR para criar nova VPC
	if stack.Spec.VpcCIDR == "" {
		return fmt.Errorf("vpcCIDR is required when existingVpcID is not specified")
	}

	// Criar VPC
	vpcName := stack.Spec.VpcName
	if vpcName == "" {
		vpcName = stack.Name + "-vpc"
	}

	createOutput, err := ec2Client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String(stack.Spec.VpcCIDR),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags:         r.buildTags(stack, vpcName),
			},
		},
	})
	if err != nil {
		return err
	}

	vpcID := aws.ToString(createOutput.Vpc.VpcId)

	// Habilitar DNS
	if stack.Spec.EnableDNSHostnames {
		_, err = ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
			VpcId:              aws.String(vpcID),
			EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		if err != nil {
			return err
		}
	}

	if stack.Spec.EnableDNSSupport {
		_, err = ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
			VpcId:            aws.String(vpcID),
			EnableDnsSupport: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		if err != nil {
			return err
		}
	}

	stack.Status.VPC = &infrav1alpha1.VPCStatusInfo{
		ID:    vpcID,
		CIDR:  stack.Spec.VpcCIDR,
		State: string(createOutput.Vpc.State),
	}

	return nil
}

// reconcileInternetGateway cria ou atualiza o Internet Gateway
func (r *ComputeStackReconciler) reconcileInternetGateway(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) error {
	logger := log.FromContext(ctx)

	// Verificar se IGW já existe no status
	if stack.Status.InternetGateway != nil && stack.Status.InternetGateway.ID != "" {
		return nil
	}

	// Se foi especificado um IGW ID existente, usar ao invés de criar
	if stack.Spec.ExistingInternetGatewayID != "" {
		logger.Info("Usando Internet Gateway existente", "igwID", stack.Spec.ExistingInternetGatewayID)

		// Verificar se o IGW existe na AWS
		output, err := ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
			InternetGatewayIds: []string{stack.Spec.ExistingInternetGatewayID},
		})
		if err != nil {
			return fmt.Errorf("existing Internet Gateway %s not found: %w", stack.Spec.ExistingInternetGatewayID, err)
		}
		if len(output.InternetGateways) == 0 {
			return fmt.Errorf("existing Internet Gateway %s not found", stack.Spec.ExistingInternetGatewayID)
		}

		igw := output.InternetGateways[0]
		state := "detached"
		for _, attachment := range igw.Attachments {
			if aws.ToString(attachment.VpcId) == stack.Status.VPC.ID {
				state = string(attachment.State)
				break
			}
		}

		stack.Status.InternetGateway = &infrav1alpha1.IGWStatusInfo{
			ID:    stack.Spec.ExistingInternetGatewayID,
			State: state,
		}
		return nil
	}

	// Criar Internet Gateway
	igwName := stack.Name + "-igw"
	createOutput, err := ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInternetGateway,
				Tags:         r.buildTags(stack, igwName),
			},
		},
	})
	if err != nil {
		return err
	}

	igwID := aws.ToString(createOutput.InternetGateway.InternetGatewayId)

	// Anexar à VPC
	_, err = ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
		VpcId:             aws.String(stack.Status.VPC.ID),
	})
	if err != nil {
		return err
	}

	stack.Status.InternetGateway = &infrav1alpha1.IGWStatusInfo{
		ID:    igwID,
		State: "attached",
	}

	return nil
}

// reconcileSubnets cria as subnets públicas e privadas
func (r *ComputeStackReconciler) reconcileSubnets(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) error {
	logger := log.FromContext(ctx)

	// Se foram especificadas subnets existentes, usar ao invés de criar
	if len(stack.Spec.ExistingSubnetIDs) > 0 {
		logger.Info("Usando Subnets existentes", "count", len(stack.Spec.ExistingSubnetIDs))

		// Verificar se as subnets existem na AWS
		output, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: stack.Spec.ExistingSubnetIDs,
		})
		if err != nil {
			return fmt.Errorf("existing subnets not found: %w", err)
		}

		// Limpar subnets existentes no status
		stack.Status.PublicSubnets = nil
		stack.Status.PrivateSubnets = nil

		// Categorizar subnets como públicas ou privadas baseado no MapPublicIpOnLaunch
		for _, subnet := range output.Subnets {
			subnetInfo := infrav1alpha1.SubnetStatusInfo{
				ID:               aws.ToString(subnet.SubnetId),
				CIDR:             aws.ToString(subnet.CidrBlock),
				AvailabilityZone: aws.ToString(subnet.AvailabilityZone),
				State:            string(subnet.State),
			}

			// Determinar se é pública ou privada
			if subnet.MapPublicIpOnLaunch != nil && *subnet.MapPublicIpOnLaunch {
				subnetInfo.Type = "public"
				stack.Status.PublicSubnets = append(stack.Status.PublicSubnets, subnetInfo)
			} else {
				subnetInfo.Type = "private"
				stack.Status.PrivateSubnets = append(stack.Status.PrivateSubnets, subnetInfo)
			}
		}
		return nil
	}

	// Se nenhuma subnet foi especificada, criar automaticamente uma subnet pública
	// Isso é necessário para o bastion instance funcionar
	if len(stack.Spec.PublicSubnets) == 0 && len(stack.Spec.PrivateSubnets) == 0 {
		logger.Info("Nenhuma subnet especificada, criando subnet pública automaticamente")

		// Verificar se já temos subnet no status (já foi criada anteriormente)
		if len(stack.Status.PublicSubnets) > 0 && stack.Status.PublicSubnets[0].ID != "" {
			logger.Info("Subnet pública automática já existe", "subnetID", stack.Status.PublicSubnets[0].ID)
			return nil
		}

		// Obter AZs disponíveis na região
		azOutput, err := ec2Client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("state"),
					Values: []string{"available"},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to get availability zones: %w", err)
		}

		if len(azOutput.AvailabilityZones) == 0 {
			return fmt.Errorf("no availability zones found")
		}

		// Usar a primeira AZ disponível
		az := aws.ToString(azOutput.AvailabilityZones[0].ZoneName)

		// Calcular CIDR da subnet baseado no VPC CIDR
		// Se VPC é /16, criar subnet /24
		// Ex: 10.201.0.0/16 -> 10.201.1.0/24
		vpcCIDR := stack.Status.VPC.CIDR
		subnetCIDR := r.calculateSubnetCIDR(vpcCIDR, 1) // índice 1 para primeira subnet

		subnetName := fmt.Sprintf("%s-public-auto-%s", stack.Name, az)

		logger.Info("Criando subnet pública automática", "cidr", subnetCIDR, "az", az, "name", subnetName)

		createOutput, err := ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
			VpcId:            aws.String(stack.Status.VPC.ID),
			CidrBlock:        aws.String(subnetCIDR),
			AvailabilityZone: aws.String(az),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSubnet,
					Tags:         r.buildTags(stack, subnetName),
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create auto subnet: %w", err)
		}

		// Habilitar auto-assign de IP público
		subnetID := aws.ToString(createOutput.Subnet.SubnetId)
		_, err = ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
			SubnetId:            aws.String(subnetID),
			MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("failed to enable auto-assign public IP: %w", err)
		}

		stack.Status.PublicSubnets = append(stack.Status.PublicSubnets, infrav1alpha1.SubnetStatusInfo{
			ID:               subnetID,
			CIDR:             subnetCIDR,
			AvailabilityZone: az,
			Type:             "public",
			State:            string(createOutput.Subnet.State),
		})

		logger.Info("Subnet pública automática criada", "subnetID", subnetID, "cidr", subnetCIDR)
		return nil
	}

	// Criar subnets públicas
	for i, subnetConfig := range stack.Spec.PublicSubnets {
		// Verificar se já existe
		if len(stack.Status.PublicSubnets) > i && stack.Status.PublicSubnets[i].ID != "" {
			continue
		}

		subnetName := subnetConfig.Name
		if subnetName == "" {
			subnetName = fmt.Sprintf("%s-public-%s", stack.Name, subnetConfig.AvailabilityZone)
		}

		createOutput, err := ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
			VpcId:            aws.String(stack.Status.VPC.ID),
			CidrBlock:        aws.String(subnetConfig.CIDR),
			AvailabilityZone: aws.String(subnetConfig.AvailabilityZone),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSubnet,
					Tags:         r.buildTags(stack, subnetName),
				},
			},
		})
		if err != nil {
			return err
		}

		// Habilitar auto-assign de IP público
		subnetID := aws.ToString(createOutput.Subnet.SubnetId)
		_, err = ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
			SubnetId:            aws.String(subnetID),
			MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		if err != nil {
			return err
		}

		stack.Status.PublicSubnets = append(stack.Status.PublicSubnets, infrav1alpha1.SubnetStatusInfo{
			ID:               subnetID,
			CIDR:             subnetConfig.CIDR,
			AvailabilityZone: subnetConfig.AvailabilityZone,
			Type:             "public",
			State:            string(createOutput.Subnet.State),
		})
	}

	// Criar subnets privadas
	for i, subnetConfig := range stack.Spec.PrivateSubnets {
		// Verificar se já existe
		if len(stack.Status.PrivateSubnets) > i && stack.Status.PrivateSubnets[i].ID != "" {
			continue
		}

		subnetName := subnetConfig.Name
		if subnetName == "" {
			subnetName = fmt.Sprintf("%s-private-%s", stack.Name, subnetConfig.AvailabilityZone)
		}

		createOutput, err := ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
			VpcId:            aws.String(stack.Status.VPC.ID),
			CidrBlock:        aws.String(subnetConfig.CIDR),
			AvailabilityZone: aws.String(subnetConfig.AvailabilityZone),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSubnet,
					Tags:         r.buildTags(stack, subnetName),
				},
			},
		})
		if err != nil {
			return err
		}

		stack.Status.PrivateSubnets = append(stack.Status.PrivateSubnets, infrav1alpha1.SubnetStatusInfo{
			ID:               aws.ToString(createOutput.Subnet.SubnetId),
			CIDR:             subnetConfig.CIDR,
			AvailabilityZone: subnetConfig.AvailabilityZone,
			Type:             "private",
			State:            string(createOutput.Subnet.State),
		})
	}

	return nil
}

// createNATGateways cria os NAT Gateways (sem aguardar)
func (r *ComputeStackReconciler) createNATGateways(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) error {
	logger := log.FromContext(ctx)

	// Se já existem NAT Gateways no status, verificar estado
	if len(stack.Status.NATGateways) > 0 {
		return nil
	}

	// Se foram especificados NAT Gateway IDs existentes, usar ao invés de criar
	if len(stack.Spec.ExistingNATGatewayIDs) > 0 {
		logger.Info("Usando NAT Gateways existentes", "count", len(stack.Spec.ExistingNATGatewayIDs))

		// Verificar se os NAT Gateways existem na AWS
		output, err := ec2Client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: stack.Spec.ExistingNATGatewayIDs,
		})
		if err != nil {
			return fmt.Errorf("existing NAT Gateways not found: %w", err)
		}

		for _, nat := range output.NatGateways {
			var elasticIP, allocationID string
			if len(nat.NatGatewayAddresses) > 0 {
				elasticIP = aws.ToString(nat.NatGatewayAddresses[0].PublicIp)
				allocationID = aws.ToString(nat.NatGatewayAddresses[0].AllocationId)
			}

			stack.Status.NATGateways = append(stack.Status.NATGateways, infrav1alpha1.NATGatewayStatusInfo{
				ID:           aws.ToString(nat.NatGatewayId),
				ElasticIP:    elasticIP,
				AllocationID: allocationID,
				SubnetID:     aws.ToString(nat.SubnetId),
				State:        string(nat.State),
			})
		}
		return nil
	}

	// Determinar quantos NAT Gateways criar
	numNATs := 1
	if stack.Spec.NATGateway.HighAvailability {
		numNATs = len(stack.Status.PublicSubnets)
	}

	for i := 0; i < numNATs; i++ {
		publicSubnet := stack.Status.PublicSubnets[i]

		// Criar Elastic IP
		eipName := fmt.Sprintf("%s-nat-eip-%d", stack.Name, i+1)
		eipOutput, err := ec2Client.AllocateAddress(ctx, &ec2.AllocateAddressInput{
			Domain: types.DomainTypeVpc,
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeElasticIp,
					Tags:         r.buildTags(stack, eipName),
				},
			},
		})
		if err != nil {
			return err
		}

		// Criar NAT Gateway
		natName := fmt.Sprintf("%s-nat-%d", stack.Name, i+1)
		natOutput, err := ec2Client.CreateNatGateway(ctx, &ec2.CreateNatGatewayInput{
			SubnetId:     aws.String(publicSubnet.ID),
			AllocationId: eipOutput.AllocationId,
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeNatgateway,
					Tags:         r.buildTags(stack, natName),
				},
			},
		})
		if err != nil {
			return err
		}

		stack.Status.NATGateways = append(stack.Status.NATGateways, infrav1alpha1.NATGatewayStatusInfo{
			ID:           aws.ToString(natOutput.NatGateway.NatGatewayId),
			ElasticIP:    aws.ToString(eipOutput.PublicIp),
			AllocationID: aws.ToString(eipOutput.AllocationId),
			SubnetID:     publicSubnet.ID,
			State:        string(natOutput.NatGateway.State),
		})
	}

	return nil
}

// checkNATGatewaysReady verifica se todos os NAT Gateways estão disponíveis
func (r *ComputeStackReconciler) checkNATGatewaysReady(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) (bool, error) {
	if len(stack.Status.NATGateways) == 0 {
		return true, nil
	}

	var natIDs []string
	for _, nat := range stack.Status.NATGateways {
		natIDs = append(natIDs, nat.ID)
	}

	output, err := ec2Client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: natIDs,
	})
	if err != nil {
		return false, err
	}

	allReady := true
	for i, nat := range output.NatGateways {
		state := string(nat.State)
		if i < len(stack.Status.NATGateways) {
			stack.Status.NATGateways[i].State = state
		}
		if nat.State != types.NatGatewayStateAvailable {
			allReady = false
		}
	}

	return allReady, nil
}

// reconcileRouteTables cria as route tables
func (r *ComputeStackReconciler) reconcileRouteTables(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) error {
	logger := log.FromContext(ctx)

	// Se já existem route tables no status, retornar
	if len(stack.Status.RouteTables) > 0 {
		return nil
	}

	// Se foram especificadas Route Table IDs existentes, usar ao invés de criar
	if len(stack.Spec.ExistingRouteTableIDs) > 0 {
		logger.Info("Usando Route Tables existentes", "count", len(stack.Spec.ExistingRouteTableIDs))

		// Verificar se as Route Tables existem na AWS
		output, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			RouteTableIds: stack.Spec.ExistingRouteTableIDs,
		})
		if err != nil {
			return fmt.Errorf("existing Route Tables not found: %w", err)
		}

		for _, rt := range output.RouteTables {
			// Determinar tipo baseado nas rotas
			rtType := "private"
			for _, route := range rt.Routes {
				if aws.ToString(route.GatewayId) != "" && aws.ToString(route.GatewayId) != "local" {
					rtType = "public"
					break
				}
			}

			// Coletar subnets associadas
			var associatedSubnets []string
			for _, assoc := range rt.Associations {
				if assoc.SubnetId != nil {
					associatedSubnets = append(associatedSubnets, aws.ToString(assoc.SubnetId))
				}
			}

			stack.Status.RouteTables = append(stack.Status.RouteTables, infrav1alpha1.RouteTableStatusInfo{
				ID:                aws.ToString(rt.RouteTableId),
				Type:              rtType,
				AssociatedSubnets: associatedSubnets,
			})
		}
		return nil
	}

	// Criar route table pública
	publicRTName := stack.Name + "-public-rt"
	publicRTOutput, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		VpcId: aws.String(stack.Status.VPC.ID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeRouteTable,
				Tags:         r.buildTags(stack, publicRTName),
			},
		},
	})
	if err != nil {
		return err
	}

	publicRTID := aws.ToString(publicRTOutput.RouteTable.RouteTableId)

	// Adicionar rota para Internet Gateway
	_, err = ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         aws.String(publicRTID),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(stack.Status.InternetGateway.ID),
	})
	if err != nil {
		return err
	}

	// Associar subnets públicas
	var publicSubnetIDs []string
	for _, subnet := range stack.Status.PublicSubnets {
		_, err = ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
			RouteTableId: aws.String(publicRTID),
			SubnetId:     aws.String(subnet.ID),
		})
		if err != nil {
			return err
		}
		publicSubnetIDs = append(publicSubnetIDs, subnet.ID)
	}

	stack.Status.RouteTables = append(stack.Status.RouteTables, infrav1alpha1.RouteTableStatusInfo{
		ID:                publicRTID,
		Type:              "public",
		AssociatedSubnets: publicSubnetIDs,
	})

	// Criar route tables privadas (uma por AZ se há NAT HA, ou uma compartilhada)
	if len(stack.Status.PrivateSubnets) > 0 && len(stack.Status.NATGateways) > 0 {
		if stack.Spec.NATGateway.HighAvailability {
			// Uma route table por AZ
			for i, nat := range stack.Status.NATGateways {
				privateRTName := fmt.Sprintf("%s-private-rt-%d", stack.Name, i+1)
				privateRTOutput, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
					VpcId: aws.String(stack.Status.VPC.ID),
					TagSpecifications: []types.TagSpecification{
						{
							ResourceType: types.ResourceTypeRouteTable,
							Tags:         r.buildTags(stack, privateRTName),
						},
					},
				})
				if err != nil {
					return err
				}

				privateRTID := aws.ToString(privateRTOutput.RouteTable.RouteTableId)

				// Rota para NAT Gateway
				_, err = ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
					RouteTableId:         aws.String(privateRTID),
					DestinationCidrBlock: aws.String("0.0.0.0/0"),
					NatGatewayId:         aws.String(nat.ID),
				})
				if err != nil {
					return err
				}

				// Associar subnet privada correspondente
				if i < len(stack.Status.PrivateSubnets) {
					subnet := stack.Status.PrivateSubnets[i]
					_, err = ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
						RouteTableId: aws.String(privateRTID),
						SubnetId:     aws.String(subnet.ID),
					})
					if err != nil {
						return err
					}

					stack.Status.RouteTables = append(stack.Status.RouteTables, infrav1alpha1.RouteTableStatusInfo{
						ID:                privateRTID,
						Type:              "private",
						AssociatedSubnets: []string{subnet.ID},
					})
				}
			}
		} else {
			// Uma route table compartilhada
			privateRTName := stack.Name + "-private-rt"
			privateRTOutput, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
				VpcId: aws.String(stack.Status.VPC.ID),
				TagSpecifications: []types.TagSpecification{
					{
						ResourceType: types.ResourceTypeRouteTable,
						Tags:         r.buildTags(stack, privateRTName),
					},
				},
			})
			if err != nil {
				return err
			}

			privateRTID := aws.ToString(privateRTOutput.RouteTable.RouteTableId)

			// Rota para NAT Gateway
			_, err = ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
				RouteTableId:         aws.String(privateRTID),
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				NatGatewayId:         aws.String(stack.Status.NATGateways[0].ID),
			})
			if err != nil {
				return err
			}

			// Associar todas as subnets privadas
			var privateSubnetIDs []string
			for _, subnet := range stack.Status.PrivateSubnets {
				_, err = ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
					RouteTableId: aws.String(privateRTID),
					SubnetId:     aws.String(subnet.ID),
				})
				if err != nil {
					return err
				}
				privateSubnetIDs = append(privateSubnetIDs, subnet.ID)
			}

			stack.Status.RouteTables = append(stack.Status.RouteTables, infrav1alpha1.RouteTableStatusInfo{
				ID:                privateRTID,
				Type:              "private",
				AssociatedSubnets: privateSubnetIDs,
			})
		}
	}

	return nil
}

// reconcileSecurityGroups cria os security groups
func (r *ComputeStackReconciler) reconcileSecurityGroups(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) error {
	logger := log.FromContext(ctx)

	// Se já existem security groups no status, retornar
	if len(stack.Status.SecurityGroups) > 0 {
		return nil
	}

	// Se foram especificados Security Group IDs existentes, usar ao invés de criar
	if len(stack.Spec.ExistingSecurityGroupIDs) > 0 {
		logger.Info("Usando Security Groups existentes", "count", len(stack.Spec.ExistingSecurityGroupIDs))

		// Verificar se os Security Groups existem na AWS
		output, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			GroupIds: stack.Spec.ExistingSecurityGroupIDs,
		})
		if err != nil {
			return fmt.Errorf("existing Security Groups not found: %w", err)
		}

		for _, sg := range output.SecurityGroups {
			stack.Status.SecurityGroups = append(stack.Status.SecurityGroups, infrav1alpha1.SecurityGroupStatusInfo{
				ID:   aws.ToString(sg.GroupId),
				Name: aws.ToString(sg.GroupName),
			})
		}
		return nil
	}

	for _, sgConfig := range stack.Spec.DefaultSecurityGroups {
		sgName := fmt.Sprintf("%s-%s", stack.Name, sgConfig.Name)
		description := sgConfig.Description
		if description == "" {
			description = fmt.Sprintf("Security group %s for ComputeStack %s", sgConfig.Name, stack.Name)
		}

		createOutput, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(sgName),
			Description: aws.String(description),
			VpcId:       aws.String(stack.Status.VPC.ID),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSecurityGroup,
					Tags:         r.buildTags(stack, sgName),
				},
			},
		})
		if err != nil {
			return err
		}

		sgID := aws.ToString(createOutput.GroupId)

		// Adicionar regras de ingress
		for _, rule := range sgConfig.IngressRules {
			toPort := rule.ToPort
			if toPort == 0 {
				toPort = rule.Port
			}

			_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
				GroupId: aws.String(sgID),
				IpPermissions: []types.IpPermission{
					{
						IpProtocol: aws.String(rule.Protocol),
						FromPort:   aws.Int32(rule.Port),
						ToPort:     aws.Int32(toPort),
						IpRanges: []types.IpRange{
							{
								CidrIp:      aws.String(rule.CIDR),
								Description: aws.String(rule.Description),
							},
						},
					},
				},
			})
			if err != nil {
				return err
			}
		}

		// Adicionar regras de egress (além da regra padrão)
		for _, rule := range sgConfig.EgressRules {
			toPort := rule.ToPort
			if toPort == 0 {
				toPort = rule.Port
			}

			_, err = ec2Client.AuthorizeSecurityGroupEgress(ctx, &ec2.AuthorizeSecurityGroupEgressInput{
				GroupId: aws.String(sgID),
				IpPermissions: []types.IpPermission{
					{
						IpProtocol: aws.String(rule.Protocol),
						FromPort:   aws.Int32(rule.Port),
						ToPort:     aws.Int32(toPort),
						IpRanges: []types.IpRange{
							{
								CidrIp:      aws.String(rule.CIDR),
								Description: aws.String(rule.Description),
							},
						},
					},
				},
			})
			if err != nil {
				// Ignorar erro se regra já existe
				continue
			}
		}

		stack.Status.SecurityGroups = append(stack.Status.SecurityGroups, infrav1alpha1.SecurityGroupStatusInfo{
			ID:   sgID,
			Name: sgName,
		})
	}

	return nil
}

// isNotFoundError checks if the error is an AWS "NotFound" type (resource already deleted)
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "NotFound") ||
		strings.Contains(errStr, "does not exist") ||
		strings.Contains(errStr, "InvalidInstanceID") ||
		strings.Contains(errStr, "InvalidGroup.NotFound") ||
		strings.Contains(errStr, "InvalidSubnetID.NotFound") ||
		strings.Contains(errStr, "InvalidRouteTableID.NotFound") ||
		strings.Contains(errStr, "InvalidInternetGatewayID.NotFound") ||
		strings.Contains(errStr, "InvalidVpcID.NotFound") ||
		strings.Contains(errStr, "InvalidNatGatewayID.NotFound") ||
		strings.Contains(errStr, "InvalidAllocationID.NotFound")
}

// deleteStack deletes all stack resources in reverse order
// IMPORTANT: Existing resources (specified via existing*ID) are NOT deleted
// Deletion follows reverse order of creation to respect dependencies:
// 1. Bastion Instance (first - uses SG and Subnet)
// 2. Bastion Security Group (after EC2 terminates)
// 3. Route Tables (need subnets disassociated)
// 4. NAT Gateways + EIPs (use subnets)
// 5. Subnets (after NAT/RT deleted)
// 6. Internet Gateway (after subnets and before VPC)
// 7. Security Groups (may have ENI dependencies)
// 8. VPC (last - everything must be deleted)
func (r *ComputeStackReconciler) deleteStack(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) error {
	logger := log.FromContext(ctx)

	if stack.Spec.DeletionPolicy == "Retain" {
		logger.Info("DeletionPolicy is Retain, keeping resources in AWS")
		return nil
	}

	stack.Status.Phase = PhaseDeleting

	// =========================================================================
	// STEP 0: Delete ALL Bastion Instances (FIRST - uses SG and Subnet)
	// Important: Find instances by VPC+Tag, not just status ID (which may be stale)
	// =========================================================================
	vpcID := ""
	if stack.Status.VPC != nil && stack.Status.VPC.ID != "" {
		vpcID = stack.Status.VPC.ID
	} else if stack.Spec.ExistingVpcID != "" {
		vpcID = stack.Spec.ExistingVpcID
	}

	if vpcID != "" {
		bastionName := fmt.Sprintf("%s-bastion", stack.Name)
		logger.Info("STEP 0: Finding Bastion Instances by tag in VPC", "vpc", vpcID, "bastionName", bastionName)

		// Find ALL instances in this VPC with the bastion Name tag that are not terminated
		descOut, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			Filters: []types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
				{Name: aws.String("tag:Name"), Values: []string{bastionName}},
				{Name: aws.String("instance-state-name"), Values: []string{
					string(types.InstanceStateNamePending),
					string(types.InstanceStateNameRunning),
					string(types.InstanceStateNameStopping),
					string(types.InstanceStateNameStopped),
					string(types.InstanceStateNameShuttingDown),
				}},
			},
		})

		if err != nil && !isNotFoundError(err) {
			logger.Error(err, "Error finding Bastion Instances in VPC")
			return fmt.Errorf("error finding bastion instances in VPC %s: %w", vpcID, err)
		}

		// Collect all non-terminated instances
		var instancesToTerminate []string
		if descOut != nil {
			for _, res := range descOut.Reservations {
				for _, inst := range res.Instances {
					if inst.InstanceId != nil && inst.State != nil &&
						inst.State.Name != types.InstanceStateNameTerminated {
						instancesToTerminate = append(instancesToTerminate, *inst.InstanceId)
					}
				}
			}
		}

		// Also add the instance from status if not already in the list
		if stack.Status.BastionInstance != nil && stack.Status.BastionInstance.ID != "" {
			statusInstanceID := stack.Status.BastionInstance.ID
			found := false
			for _, id := range instancesToTerminate {
				if id == statusInstanceID {
					found = true
					break
				}
			}
			if !found {
				instancesToTerminate = append(instancesToTerminate, statusInstanceID)
			}
		}

		if len(instancesToTerminate) > 0 {
			logger.Info("Found Bastion Instances to terminate", "count", len(instancesToTerminate), "ids", instancesToTerminate)

			// Terminate all instances
			_, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
				InstanceIds: instancesToTerminate,
			})
			if err != nil && !isNotFoundError(err) {
				logger.Error(err, "Failed to terminate Bastion Instances", "ids", instancesToTerminate)
				return fmt.Errorf("failed to terminate bastion instances %v: %w", instancesToTerminate, err)
			}

			// Wait for ALL instances to terminate COMPLETELY before continuing
			for _, instanceID := range instancesToTerminate {
				logger.Info("Waiting for Bastion Instance to terminate completely...", "id", instanceID)
				waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
				err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
					InstanceIds: []string{instanceID},
				}, 5*time.Minute)
				if err != nil && !isNotFoundError(err) {
					logger.Error(err, "Timeout waiting for Bastion Instance to terminate", "id", instanceID)
					return fmt.Errorf("timeout waiting for bastion instance %s to terminate: %w", instanceID, err)
				}
				logger.Info("Bastion Instance terminated successfully", "id", instanceID)
			}
		} else {
			logger.Info("No Bastion Instances found to terminate")
		}

		stack.Status.BastionInstance = nil
	} else if stack.Status.BastionInstance != nil && stack.Status.BastionInstance.ID != "" {
		// Fallback: No VPC ID available, try to terminate by status ID only
		instanceID := stack.Status.BastionInstance.ID
		logger.Info("STEP 0: Terminating Bastion Instance (no VPC, using status ID)", "id", instanceID)

		descOut, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		})

		if err != nil && isNotFoundError(err) {
			logger.Info("Bastion Instance already deleted", "id", instanceID)
			stack.Status.BastionInstance = nil
		} else if err != nil {
			logger.Error(err, "Error checking Bastion Instance", "id", instanceID)
			return fmt.Errorf("error checking bastion instance %s: %w", instanceID, err)
		} else if len(descOut.Reservations) > 0 && len(descOut.Reservations[0].Instances) > 0 {
			state := descOut.Reservations[0].Instances[0].State.Name
			if state != types.InstanceStateNameTerminated {
				_, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
					InstanceIds: []string{instanceID},
				})
				if err != nil && !isNotFoundError(err) {
					logger.Error(err, "Failed to terminate Bastion Instance", "id", instanceID)
					return fmt.Errorf("failed to terminate bastion instance %s: %w", instanceID, err)
				}

				logger.Info("Waiting for Bastion Instance to terminate completely...", "id", instanceID)
				waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
				err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
					InstanceIds: []string{instanceID},
				}, 5*time.Minute)
				if err != nil && !isNotFoundError(err) {
					logger.Error(err, "Timeout waiting for Bastion Instance to terminate", "id", instanceID)
					return fmt.Errorf("timeout waiting for bastion instance %s to terminate: %w", instanceID, err)
				}
				logger.Info("Bastion Instance terminated successfully", "id", instanceID)
			}
			stack.Status.BastionInstance = nil
		}
	}

	// =========================================================================
	// STEP 1: Delete Bastion Security Group (after EC2 terminates)
	// =========================================================================
	if stack.Status.BastionSecurityGroup != nil && stack.Status.BastionSecurityGroup.ID != "" {
		sgID := stack.Status.BastionSecurityGroup.ID
		logger.Info("STEP 1: Deleting Bastion Security Group", "id", sgID)
		_, err := ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sgID),
		})
		if err != nil && !isNotFoundError(err) {
			// If failed due to DependencyViolation, EC2 may not have released the ENI yet
			if strings.Contains(err.Error(), "DependencyViolation") {
				logger.Info("Bastion SG still has dependencies, waiting...", "id", sgID)
				return fmt.Errorf("bastion security group %s still has dependencies (EC2 ENI): %w", sgID, err)
			}
			logger.Error(err, "Failed to delete Bastion Security Group", "id", sgID)
			return fmt.Errorf("failed to delete bastion security group %s: %w", sgID, err)
		}
		logger.Info("Bastion Security Group deleted", "id", sgID)
		stack.Status.BastionSecurityGroup = nil
	}

	// =========================================================================
	// STEP 2: Disassociate and Delete Route Tables
	// =========================================================================
	if len(stack.Spec.ExistingRouteTableIDs) == 0 && len(stack.Status.RouteTables) > 0 {
		for i, rt := range stack.Status.RouteTables {
			logger.Info("STEP 2: Deleting Route Table", "id", rt.ID, "index", i)

			// First, disassociate the route table from subnets
			descOut, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
				RouteTableIds: []string{rt.ID},
			})
			if err != nil && isNotFoundError(err) {
				logger.Info("Route Table already deleted", "id", rt.ID)
				continue
			} else if err != nil {
				logger.Error(err, "Error describing Route Table", "id", rt.ID)
				return fmt.Errorf("error describing route table %s: %w", rt.ID, err)
			}

			// Disassociate all associations (except main)
			if len(descOut.RouteTables) > 0 {
				for _, assoc := range descOut.RouteTables[0].Associations {
					if assoc.Main != nil && *assoc.Main {
						continue // Cannot disassociate main route table
					}
					if assoc.RouteTableAssociationId != nil {
						logger.Info("Disassociating Route Table", "associationId", *assoc.RouteTableAssociationId)
						_, err := ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
							AssociationId: assoc.RouteTableAssociationId,
						})
						if err != nil && !isNotFoundError(err) {
							logger.Error(err, "Failed to disassociate Route Table", "associationId", *assoc.RouteTableAssociationId)
							return fmt.Errorf("failed to disassociate route table: %w", err)
						}
					}
				}
			}

			// Now delete the route table
			_, err = ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
				RouteTableId: aws.String(rt.ID),
			})
			if err != nil && !isNotFoundError(err) {
				if strings.Contains(err.Error(), "DependencyViolation") {
					logger.Info("Route Table still has dependencies", "id", rt.ID)
					return fmt.Errorf("route table %s still has dependencies: %w", rt.ID, err)
				}
				logger.Error(err, "Failed to delete Route Table", "id", rt.ID)
				return fmt.Errorf("failed to delete route table %s: %w", rt.ID, err)
			}
			logger.Info("Route Table deleted", "id", rt.ID)
		}
		stack.Status.RouteTables = nil
	}

	// =========================================================================
	// STEP 3: Delete NAT Gateways and Elastic IPs
	// =========================================================================
	if len(stack.Spec.ExistingNATGatewayIDs) == 0 && len(stack.Status.NATGateways) > 0 {
		for _, nat := range stack.Status.NATGateways {
			logger.Info("STEP 3: Deleting NAT Gateway", "id", nat.ID)
			_, err := ec2Client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
				NatGatewayId: aws.String(nat.ID),
			})
			if err != nil && !isNotFoundError(err) {
				logger.Error(err, "Failed to delete NAT Gateway", "id", nat.ID)
				return fmt.Errorf("failed to delete NAT gateway %s: %w", nat.ID, err)
			}
		}

		// Wait for NAT Gateways to be deleted before releasing EIPs
		for _, nat := range stack.Status.NATGateways {
			logger.Info("Waiting for NAT Gateway to be deleted...", "id", nat.ID)
			waiter := ec2.NewNatGatewayDeletedWaiter(ec2Client)
			err := waiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{
				NatGatewayIds: []string{nat.ID},
			}, 5*time.Minute)
			if err != nil && !isNotFoundError(err) {
				logger.Error(err, "Timeout waiting for NAT Gateway to delete", "id", nat.ID)
				return fmt.Errorf("timeout waiting for NAT gateway %s: %w", nat.ID, err)
			}

			// Release Elastic IP
			if nat.AllocationID != "" {
				logger.Info("Releasing Elastic IP", "allocationID", nat.AllocationID)
				_, err := ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
					AllocationId: aws.String(nat.AllocationID),
				})
				if err != nil && !isNotFoundError(err) {
					logger.Error(err, "Failed to release Elastic IP", "allocationID", nat.AllocationID)
					return fmt.Errorf("failed to release elastic IP %s: %w", nat.AllocationID, err)
				}
			}
		}
		stack.Status.NATGateways = nil
	}

	// =========================================================================
	// STEP 4: Delete Subnets
	// =========================================================================
	if len(stack.Spec.ExistingSubnetIDs) == 0 {
		allSubnets := append(stack.Status.PublicSubnets, stack.Status.PrivateSubnets...)
		for _, subnet := range allSubnets {
			logger.Info("STEP 4: Deleting Subnet", "id", subnet.ID)
			_, err := ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
				SubnetId: aws.String(subnet.ID),
			})
			if err != nil && !isNotFoundError(err) {
				if strings.Contains(err.Error(), "DependencyViolation") {
					logger.Info("Subnet still has dependencies", "id", subnet.ID)
					return fmt.Errorf("subnet %s still has dependencies: %w", subnet.ID, err)
				}
				logger.Error(err, "Failed to delete Subnet", "id", subnet.ID)
				return fmt.Errorf("failed to delete subnet %s: %w", subnet.ID, err)
			}
			logger.Info("Subnet deleted", "id", subnet.ID)
		}
		stack.Status.PublicSubnets = nil
		stack.Status.PrivateSubnets = nil
	}

	// =========================================================================
	// STEP 5: Detach and Delete Internet Gateway
	// =========================================================================
	if stack.Spec.ExistingInternetGatewayID == "" {
		if stack.Status.InternetGateway != nil && stack.Status.InternetGateway.ID != "" {
			igwID := stack.Status.InternetGateway.ID
			vpcID := ""
			if stack.Status.VPC != nil {
				vpcID = stack.Status.VPC.ID
			}

			logger.Info("STEP 5: Detaching Internet Gateway", "id", igwID, "vpcId", vpcID)

			if vpcID != "" {
				_, err := ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
					InternetGatewayId: aws.String(igwID),
					VpcId:             aws.String(vpcID),
				})
				if err != nil && !isNotFoundError(err) {
					if strings.Contains(err.Error(), "DependencyViolation") {
						logger.Info("IGW still has dependencies (mapped public IPs)", "id", igwID)
						return fmt.Errorf("internet gateway %s still has dependencies: %w", igwID, err)
					}
					// Gateway.NotAttached is not a fatal error
					if !strings.Contains(err.Error(), "Gateway.NotAttached") {
						logger.Error(err, "Failed to detach Internet Gateway")
						return fmt.Errorf("failed to detach internet gateway %s: %w", igwID, err)
					}
				}
			}

			logger.Info("Deleting Internet Gateway", "id", igwID)
			_, err := ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
				InternetGatewayId: aws.String(igwID),
			})
			if err != nil && !isNotFoundError(err) {
				if strings.Contains(err.Error(), "DependencyViolation") {
					logger.Info("IGW still has dependencies", "id", igwID)
					return fmt.Errorf("internet gateway %s still has dependencies: %w", igwID, err)
				}
				logger.Error(err, "Failed to delete Internet Gateway")
				return fmt.Errorf("failed to delete internet gateway %s: %w", igwID, err)
			}
			logger.Info("Internet Gateway deleted", "id", igwID)
			stack.Status.InternetGateway = nil
		}
	}

	// =========================================================================
	// STEP 6: Delete Security Groups (after subnets to release ENIs)
	// =========================================================================
	if len(stack.Spec.ExistingSecurityGroupIDs) == 0 && len(stack.Status.SecurityGroups) > 0 {
		for _, sg := range stack.Status.SecurityGroups {
			logger.Info("STEP 6: Deleting Security Group", "id", sg.ID)
			_, err := ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
				GroupId: aws.String(sg.ID),
			})
			if err != nil && !isNotFoundError(err) {
				if strings.Contains(err.Error(), "DependencyViolation") {
					logger.Info("Security Group still has dependencies", "id", sg.ID)
					return fmt.Errorf("security group %s still has dependencies: %w", sg.ID, err)
				}
				logger.Error(err, "Failed to delete Security Group", "id", sg.ID)
				return fmt.Errorf("failed to delete security group %s: %w", sg.ID, err)
			}
			logger.Info("Security Group deleted", "id", sg.ID)
		}
		stack.Status.SecurityGroups = nil
	}

	// =========================================================================
	// STEP 7: Delete VPC (LAST - everything must be deleted)
	// =========================================================================
	if stack.Spec.ExistingVpcID == "" {
		if stack.Status.VPC != nil && stack.Status.VPC.ID != "" {
			vpcID := stack.Status.VPC.ID
			logger.Info("STEP 7: Deleting VPC", "id", vpcID)
			_, err := ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
				VpcId: aws.String(vpcID),
			})
			if err != nil && !isNotFoundError(err) {
				if strings.Contains(err.Error(), "DependencyViolation") {
					logger.Info("VPC still has dependencies - checking orphan resources", "id", vpcID)
					return fmt.Errorf("VPC %s still has dependencies: %w", vpcID, err)
				}
				logger.Error(err, "Failed to delete VPC")
				return fmt.Errorf("failed to delete VPC %s: %w", vpcID, err)
			}
			logger.Info("VPC deleted successfully!", "id", vpcID)
			stack.Status.VPC = nil
		}
	}

	logger.Info("All stack resources deleted successfully!")
	return nil
}

// deleteStackAsync deletes stack resources one step at a time without blocking.
// Returns (done bool, error) - done=true when all resources are deleted.
// This function is called repeatedly by the reconcile loop until done=true.
func (r *ComputeStackReconciler) deleteStackAsync(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) (bool, error) {
	logger := log.FromContext(ctx)

	if stack.Spec.DeletionPolicy == "Retain" {
		logger.Info("DeletionPolicy is Retain, keeping resources in AWS")
		return true, nil
	}

	// Get VPC ID for instance lookup
	vpcID := ""
	if stack.Status.VPC != nil && stack.Status.VPC.ID != "" {
		vpcID = stack.Status.VPC.ID
	} else if stack.Spec.ExistingVpcID != "" {
		vpcID = stack.Spec.ExistingVpcID
	}

	// =========================================================================
	// STEP 0: Delete Bastion Instance (FIRST - uses SG and Subnet)
	// =========================================================================
	if stack.Status.BastionInstance != nil && stack.Status.BastionInstance.ID != "" {
		instanceID := stack.Status.BastionInstance.ID
		stack.Status.Message = fmt.Sprintf("Deleting Bastion Instance %s...", instanceID)
		logger.Info("STEP 0: Checking Bastion Instance", "id", instanceID)

		descOut, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		})
		if err != nil && isNotFoundError(err) {
			logger.Info("Bastion Instance already deleted", "id", instanceID)
			stack.Status.BastionInstance = nil
			return false, nil // Continue to next step
		} else if err != nil {
			return false, fmt.Errorf("error checking bastion instance: %w", err)
		}

		if len(descOut.Reservations) > 0 && len(descOut.Reservations[0].Instances) > 0 {
			state := descOut.Reservations[0].Instances[0].State.Name
			if state == types.InstanceStateNameTerminated {
				logger.Info("Bastion Instance terminated", "id", instanceID)
				stack.Status.BastionInstance = nil
				return false, nil // Continue to next step
			}
			if state != types.InstanceStateNameShuttingDown {
				logger.Info("Terminating Bastion Instance", "id", instanceID, "currentState", state)
				_, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
					InstanceIds: []string{instanceID},
				})
				if err != nil && !isNotFoundError(err) {
					return false, fmt.Errorf("failed to terminate bastion instance: %w", err)
				}
			}
			stack.Status.Message = fmt.Sprintf("Waiting for Bastion Instance %s to terminate...", instanceID)
			return false, nil // Will check again next reconcile
		}
		stack.Status.BastionInstance = nil
		return false, nil // Continue to next step
	}

	// Also check for orphan instances in VPC
	if vpcID != "" {
		bastionName := fmt.Sprintf("%s-bastion", stack.Name)
		descOut, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			Filters: []types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
				{Name: aws.String("tag:Name"), Values: []string{bastionName}},
				{Name: aws.String("instance-state-name"), Values: []string{
					string(types.InstanceStateNamePending),
					string(types.InstanceStateNameRunning),
					string(types.InstanceStateNameStopping),
					string(types.InstanceStateNameStopped),
					string(types.InstanceStateNameShuttingDown),
				}},
			},
		})
		if err == nil && descOut != nil {
			for _, res := range descOut.Reservations {
				for _, inst := range res.Instances {
					if inst.InstanceId != nil && inst.State != nil && inst.State.Name != types.InstanceStateNameTerminated {
						logger.Info("Found orphan bastion instance, terminating", "id", *inst.InstanceId)
						stack.Status.Message = fmt.Sprintf("Terminating orphan instance %s...", *inst.InstanceId)
						ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
							InstanceIds: []string{*inst.InstanceId},
						})
						return false, nil // Will check again next reconcile
					}
				}
			}
		}
	}

	// =========================================================================
	// STEP 0.5: Delete auto-generated Key Pair (if applicable)
	// =========================================================================
	// Check if there's an auto-generated key pair to delete
	keyPairName := fmt.Sprintf("%s-bastion-key", stack.Name)
	// Try to delete the key pair - it's OK if it doesn't exist
	_, err := ec2Client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyPairName),
	})
	if err != nil && !isNotFoundError(err) {
		// Log but don't fail - key pair deletion is best effort
		logger.Info("Note: Could not delete key pair (may not exist)", "keyPairName", keyPairName, "error", err.Error())
	} else if err == nil {
		logger.Info("STEP 0.5: Deleted auto-generated Key Pair", "keyPairName", keyPairName)
	}

	// =========================================================================
	// STEP 1: Delete Bastion Security Group
	// =========================================================================
	if stack.Status.BastionSecurityGroup != nil && stack.Status.BastionSecurityGroup.ID != "" {
		sgID := stack.Status.BastionSecurityGroup.ID
		stack.Status.Message = fmt.Sprintf("Deleting Bastion Security Group %s...", sgID)
		logger.Info("STEP 1: Deleting Bastion Security Group", "id", sgID)

		_, err := ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sgID),
		})
		if err != nil && isNotFoundError(err) {
			logger.Info("Bastion Security Group already deleted", "id", sgID)
			stack.Status.BastionSecurityGroup = nil
			return false, nil
		} else if err != nil {
			if strings.Contains(err.Error(), "DependencyViolation") {
				stack.Status.Message = fmt.Sprintf("Waiting for Bastion SG %s dependencies to clear...", sgID)
				return false, nil // Will retry
			}
			return false, fmt.Errorf("failed to delete bastion security group: %w", err)
		}
		logger.Info("Bastion Security Group deleted", "id", sgID)
		stack.Status.BastionSecurityGroup = nil
		return false, nil
	}

	// =========================================================================
	// STEP 2: Delete Route Tables
	// =========================================================================
	if len(stack.Spec.ExistingRouteTableIDs) == 0 && len(stack.Status.RouteTables) > 0 {
		rt := stack.Status.RouteTables[0]
		stack.Status.Message = fmt.Sprintf("Deleting Route Table %s...", rt.ID)
		logger.Info("STEP 2: Deleting Route Table", "id", rt.ID)

		// First disassociate
		descOut, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			RouteTableIds: []string{rt.ID},
		})
		if err != nil && isNotFoundError(err) {
			logger.Info("Route Table already deleted", "id", rt.ID)
			stack.Status.RouteTables = stack.Status.RouteTables[1:]
			return false, nil
		} else if err != nil {
			return false, fmt.Errorf("error describing route table: %w", err)
		}

		if len(descOut.RouteTables) > 0 {
			for _, assoc := range descOut.RouteTables[0].Associations {
				if assoc.Main != nil && *assoc.Main {
					continue
				}
				if assoc.RouteTableAssociationId != nil {
					logger.Info("Disassociating Route Table", "associationId", *assoc.RouteTableAssociationId)
					ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
						AssociationId: assoc.RouteTableAssociationId,
					})
				}
			}
		}

		_, err = ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
			RouteTableId: aws.String(rt.ID),
		})
		if err != nil && !isNotFoundError(err) {
			if strings.Contains(err.Error(), "DependencyViolation") {
				stack.Status.Message = fmt.Sprintf("Waiting for Route Table %s dependencies...", rt.ID)
				return false, nil
			}
			return false, fmt.Errorf("failed to delete route table: %w", err)
		}
		logger.Info("Route Table deleted", "id", rt.ID)
		stack.Status.RouteTables = stack.Status.RouteTables[1:]
		return false, nil
	}

	// =========================================================================
	// STEP 3: Delete NAT Gateways and Elastic IPs
	// =========================================================================
	if len(stack.Spec.ExistingNATGatewayIDs) == 0 && len(stack.Status.NATGateways) > 0 {
		nat := stack.Status.NATGateways[0]
		stack.Status.Message = fmt.Sprintf("Deleting NAT Gateway %s...", nat.ID)
		logger.Info("STEP 3: Checking NAT Gateway", "id", nat.ID)

		// Check NAT Gateway state
		descOut, err := ec2Client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []string{nat.ID},
		})
		if err != nil && isNotFoundError(err) {
			logger.Info("NAT Gateway already deleted", "id", nat.ID)
			// Release EIP if exists
			if nat.AllocationID != "" {
				ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
					AllocationId: aws.String(nat.AllocationID),
				})
			}
			stack.Status.NATGateways = stack.Status.NATGateways[1:]
			return false, nil
		}

		if len(descOut.NatGateways) > 0 {
			state := descOut.NatGateways[0].State
			if state == types.NatGatewayStateDeleted {
				logger.Info("NAT Gateway deleted", "id", nat.ID)
				if nat.AllocationID != "" {
					logger.Info("Releasing Elastic IP", "allocationID", nat.AllocationID)
					ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
						AllocationId: aws.String(nat.AllocationID),
					})
				}
				stack.Status.NATGateways = stack.Status.NATGateways[1:]
				return false, nil
			}
			if state != types.NatGatewayStateDeleting {
				logger.Info("Deleting NAT Gateway", "id", nat.ID)
				_, err := ec2Client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
					NatGatewayId: aws.String(nat.ID),
				})
				if err != nil && !isNotFoundError(err) {
					return false, fmt.Errorf("failed to delete NAT gateway: %w", err)
				}
			}
			stack.Status.Message = fmt.Sprintf("Waiting for NAT Gateway %s to delete...", nat.ID)
			return false, nil // Will check again
		}
		stack.Status.NATGateways = stack.Status.NATGateways[1:]
		return false, nil
	}

	// =========================================================================
	// STEP 4: Delete Subnets
	// =========================================================================
	if len(stack.Spec.ExistingSubnetIDs) == 0 {
		allSubnets := append(stack.Status.PublicSubnets, stack.Status.PrivateSubnets...)
		if len(allSubnets) > 0 {
			subnet := allSubnets[0]
			stack.Status.Message = fmt.Sprintf("Deleting Subnet %s...", subnet.ID)
			logger.Info("STEP 4: Deleting Subnet", "id", subnet.ID)

			_, err := ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
				SubnetId: aws.String(subnet.ID),
			})
			if err != nil && !isNotFoundError(err) {
				if strings.Contains(err.Error(), "DependencyViolation") {
					stack.Status.Message = fmt.Sprintf("Waiting for Subnet %s dependencies...", subnet.ID)
					return false, nil
				}
				return false, fmt.Errorf("failed to delete subnet: %w", err)
			}
			logger.Info("Subnet deleted", "id", subnet.ID)

			// Remove from appropriate list
			if len(stack.Status.PublicSubnets) > 0 && stack.Status.PublicSubnets[0].ID == subnet.ID {
				stack.Status.PublicSubnets = stack.Status.PublicSubnets[1:]
			} else if len(stack.Status.PrivateSubnets) > 0 {
				stack.Status.PrivateSubnets = stack.Status.PrivateSubnets[1:]
			}
			return false, nil
		}
	}

	// =========================================================================
	// STEP 5: Detach and Delete Internet Gateway
	// =========================================================================
	if stack.Spec.ExistingInternetGatewayID == "" && stack.Status.InternetGateway != nil && stack.Status.InternetGateway.ID != "" {
		igwID := stack.Status.InternetGateway.ID
		stack.Status.Message = fmt.Sprintf("Deleting Internet Gateway %s...", igwID)
		logger.Info("STEP 5: Deleting Internet Gateway", "id", igwID)

		// Detach first
		if vpcID != "" {
			_, err := ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
				InternetGatewayId: aws.String(igwID),
				VpcId:             aws.String(vpcID),
			})
			if err != nil && !isNotFoundError(err) && !strings.Contains(err.Error(), "Gateway.NotAttached") {
				if strings.Contains(err.Error(), "DependencyViolation") {
					stack.Status.Message = fmt.Sprintf("Waiting for IGW %s dependencies...", igwID)
					return false, nil
				}
				return false, fmt.Errorf("failed to detach internet gateway: %w", err)
			}
		}

		// Delete
		_, err := ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(igwID),
		})
		if err != nil && !isNotFoundError(err) {
			if strings.Contains(err.Error(), "DependencyViolation") {
				stack.Status.Message = fmt.Sprintf("Waiting for IGW %s dependencies...", igwID)
				return false, nil
			}
			return false, fmt.Errorf("failed to delete internet gateway: %w", err)
		}
		logger.Info("Internet Gateway deleted", "id", igwID)
		stack.Status.InternetGateway = nil
		return false, nil
	}

	// =========================================================================
	// STEP 6: Delete Security Groups
	// =========================================================================
	if len(stack.Spec.ExistingSecurityGroupIDs) == 0 && len(stack.Status.SecurityGroups) > 0 {
		sg := stack.Status.SecurityGroups[0]
		stack.Status.Message = fmt.Sprintf("Deleting Security Group %s...", sg.ID)
		logger.Info("STEP 6: Deleting Security Group", "id", sg.ID)

		_, err := ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sg.ID),
		})
		if err != nil && !isNotFoundError(err) {
			if strings.Contains(err.Error(), "DependencyViolation") {
				stack.Status.Message = fmt.Sprintf("Waiting for SG %s dependencies...", sg.ID)
				return false, nil
			}
			return false, fmt.Errorf("failed to delete security group: %w", err)
		}
		logger.Info("Security Group deleted", "id", sg.ID)
		stack.Status.SecurityGroups = stack.Status.SecurityGroups[1:]
		return false, nil
	}

	// =========================================================================
	// STEP 7: Delete orphan Route Tables (query AWS directly, not from status)
	// =========================================================================
	if vpcID != "" {
		// Query all route tables in VPC with ComputeStack tag (not tracked in status)
		descRT, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			Filters: []types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
				{Name: aws.String("tag:ComputeStack"), Values: []string{stack.Name}},
			},
		})
		if err != nil && !isNotFoundError(err) {
			return false, fmt.Errorf("failed to describe route tables in VPC: %w", err)
		}

		if descRT != nil && len(descRT.RouteTables) > 0 {
			for _, rt := range descRT.RouteTables {
				rtID := aws.ToString(rt.RouteTableId)

				// Skip main route table (auto-deleted with VPC)
				isMain := false
				for _, assoc := range rt.Associations {
					if assoc.Main != nil && *assoc.Main {
						isMain = true
						break
					}
				}
				if isMain {
					continue
				}

				stack.Status.Message = fmt.Sprintf("Deleting orphan Route Table %s...", rtID)
				logger.Info("STEP 7: Found orphan Route Table in VPC, deleting", "id", rtID)

				// Disassociate first
				for _, assoc := range rt.Associations {
					if assoc.RouteTableAssociationId != nil && (assoc.Main == nil || !*assoc.Main) {
						logger.Info("Disassociating orphan Route Table", "associationId", *assoc.RouteTableAssociationId)
						ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
							AssociationId: assoc.RouteTableAssociationId,
						})
					}
				}

				// Delete route table
				_, err := ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
					RouteTableId: aws.String(rtID),
				})
				if err != nil && !isNotFoundError(err) {
					if strings.Contains(err.Error(), "DependencyViolation") {
						stack.Status.Message = fmt.Sprintf("Waiting for orphan RT %s dependencies...", rtID)
						return false, nil
					}
					logger.Error(err, "Failed to delete orphan route table (will retry)", "id", rtID)
					return false, nil
				}
				logger.Info("Orphan Route Table deleted", "id", rtID)
				return false, nil // Process one at a time
			}
		}
	}

	// =========================================================================
	// STEP 8: Delete VPC (LAST)
	// =========================================================================
	if stack.Spec.ExistingVpcID == "" && stack.Status.VPC != nil && stack.Status.VPC.ID != "" {
		vpcID := stack.Status.VPC.ID
		stack.Status.Message = fmt.Sprintf("Deleting VPC %s...", vpcID)
		logger.Info("STEP 8: Deleting VPC", "id", vpcID)

		_, err := ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
			VpcId: aws.String(vpcID),
		})
		if err != nil && !isNotFoundError(err) {
			if strings.Contains(err.Error(), "DependencyViolation") {
				stack.Status.Message = fmt.Sprintf("Waiting for VPC %s dependencies...", vpcID)
				return false, nil
			}
			return false, fmt.Errorf("failed to delete VPC: %w", err)
		}
		logger.Info("VPC deleted", "id", vpcID)
		stack.Status.VPC = nil
		return false, nil
	}

	// All resources deleted!
	logger.Info("All ComputeStack resources deleted successfully!")
	stack.Status.Message = "All resources deleted"
	return true, nil
}

// buildTags constrói as tags para os recursos
func (r *ComputeStackReconciler) buildTags(stack *infrav1alpha1.ComputeStack, name string) []types.Tag {
	// Usar map para evitar tags duplicadas
	tagMap := make(map[string]string)

	// Tags padrão (podem ser sobrescritas pelas tags customizadas)
	tagMap["Name"] = name
	tagMap["ManagedBy"] = "infra-operator"
	tagMap["ComputeStack"] = stack.Name

	// Adicionar tags customizadas (sobrescrevem as padrão se necessário)
	for k, v := range stack.Spec.Tags {
		tagMap[k] = v
	}

	// Converter map para slice de tags
	tags := make([]types.Tag, 0, len(tagMap))
	for k, v := range tagMap {
		tags = append(tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	return tags
}

// calculateSubnetCIDR calcula o CIDR de uma subnet baseado no VPC CIDR e um índice
// Exemplo: VPC 10.201.0.0/16, índice 1 -> 10.201.1.0/24
func (r *ComputeStackReconciler) calculateSubnetCIDR(vpcCIDR string, index int) string {
	// Parse VPC CIDR (ex: 10.201.0.0/16)
	parts := strings.Split(vpcCIDR, "/")
	if len(parts) != 2 {
		return fmt.Sprintf("10.0.%d.0/24", index) // fallback
	}

	ipParts := strings.Split(parts[0], ".")
	if len(ipParts) != 4 {
		return fmt.Sprintf("10.0.%d.0/24", index) // fallback
	}

	// Criar subnet CIDR /24 no terceiro octeto
	// Ex: 10.201.0.0/16 -> 10.201.1.0/24 (index=1)
	return fmt.Sprintf("%s.%s.%d.0/24", ipParts[0], ipParts[1], index)
}

// reconcileBastionSecurityGroup cria o Security Group para o bastion (SSH)
func (r *ComputeStackReconciler) reconcileBastionSecurityGroup(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) error {
	logger := log.FromContext(ctx)

	// Verificar se já existe
	if stack.Status.BastionSecurityGroup != nil && stack.Status.BastionSecurityGroup.ID != "" {
		return nil
	}

	sgName := fmt.Sprintf("%s-bastion-sg", stack.Name)
	description := "Security Group for Bastion/SSH access"

	logger.Info("Creating Bastion Security Group", "name", sgName)

	createOutput, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sgName),
		Description: aws.String(description),
		VpcId:       aws.String(stack.Status.VPC.ID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags:         r.buildTags(stack, sgName),
			},
		},
	})
	if err != nil {
		return err
	}

	sgID := aws.ToString(createOutput.GroupId)

	// Adicionar regras de SSH
	sshCIDRs := stack.Spec.BastionInstance.SSHAllowedCIDRs
	if len(sshCIDRs) == 0 {
		sshCIDRs = []string{"0.0.0.0/0"} // Default: permitir de qualquer lugar
	}

	for _, cidr := range sshCIDRs {
		_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(sgID),
			IpPermissions: []types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(22),
					ToPort:     aws.Int32(22),
					IpRanges: []types.IpRange{
						{
							CidrIp:      aws.String(cidr),
							Description: aws.String(fmt.Sprintf("SSH access from %s", cidr)),
						},
					},
				},
			},
		})
		if err != nil {
			logger.Error(err, "Failed to add SSH ingress rule", "cidr", cidr)
			// Continuar mesmo com erro para tentar adicionar outras regras
		}
	}

	stack.Status.BastionSecurityGroup = &infrav1alpha1.SecurityGroupStatusInfo{
		ID:   sgID,
		Name: sgName,
	}

	logger.Info("Bastion Security Group created", "id", sgID)
	return nil
}

// reconcileBastionInstance cria a instância EC2 bastion
func (r *ComputeStackReconciler) reconcileBastionInstance(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) error {
	logger := log.FromContext(ctx)

	// Verificar se já existe
	if stack.Status.BastionInstance != nil && stack.Status.BastionInstance.ID != "" {
		return nil
	}

	bastionConfig := stack.Spec.BastionInstance

	// Nome da instância
	instanceName := bastionConfig.Name
	if instanceName == "" {
		instanceName = fmt.Sprintf("%s-bastion", stack.Name)
	}

	// Tipo de instância
	instanceType := bastionConfig.InstanceType
	if instanceType == "" {
		instanceType = "t3.micro"
	}

	// Buscar AMI Amazon Linux 2 se não especificada
	imageID := bastionConfig.ImageID
	if imageID == "" {
		// Buscar AMI mais recente do Amazon Linux 2
		amiOutput, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("name"),
					Values: []string{"amzn2-ami-hvm-*-x86_64-gp2"},
				},
				{
					Name:   aws.String("state"),
					Values: []string{"available"},
				},
				{
					Name:   aws.String("owner-alias"),
					Values: []string{"amazon"},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to find Amazon Linux 2 AMI: %w", err)
		}

		if len(amiOutput.Images) == 0 {
			return fmt.Errorf("no Amazon Linux 2 AMI found")
		}

		// Ordenar por data de criação e pegar a mais recente
		latestAMI := amiOutput.Images[0]
		for _, ami := range amiOutput.Images {
			if aws.ToString(ami.CreationDate) > aws.ToString(latestAMI.CreationDate) {
				latestAMI = ami
			}
		}
		imageID = aws.ToString(latestAMI.ImageId)
		logger.Info("Using Amazon Linux 2 AMI", "imageID", imageID)
	}

	// Subnet - usar primeira subnet pública
	var subnetID string
	if len(stack.Status.PublicSubnets) > 0 {
		subnetID = stack.Status.PublicSubnets[0].ID
	} else {
		return fmt.Errorf("no public subnet available for bastion instance")
	}

	// Security Group
	var securityGroupIDs []string
	if stack.Status.BastionSecurityGroup != nil && stack.Status.BastionSecurityGroup.ID != "" {
		securityGroupIDs = append(securityGroupIDs, stack.Status.BastionSecurityGroup.ID)
	} else {
		return fmt.Errorf("bastion security group not found")
	}

	// Volume size
	rootVolumeSize := bastionConfig.RootVolumeSize
	if rootVolumeSize == 0 {
		rootVolumeSize = 20
	}

	// Associate public IP
	associatePublicIP := true
	if bastionConfig.AssociatePublicIP {
		associatePublicIP = bastionConfig.AssociatePublicIP
	}

	// Determinar o keyName a ser usado
	keyName := bastionConfig.KeyName
	keyPairGenerated := false
	secretName := ""

	// Se keyName não foi especificado, gerar um key pair automaticamente
	if keyName == "" {
		keyPairName := fmt.Sprintf("%s-bastion-key", stack.Name)
		secretName = fmt.Sprintf("%s-ssh-key", stack.Name)

		logger.Info("No keyName specified, generating key pair automatically", "keyPairName", keyPairName)

		// Criar key pair na AWS
		createKeyOutput, err := ec2Client.CreateKeyPair(ctx, &ec2.CreateKeyPairInput{
			KeyName: aws.String(keyPairName),
			KeyType: types.KeyTypeRsa,
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeKeyPair,
					Tags:         r.buildTags(stack, keyPairName),
				},
			},
		})
		if err != nil {
			// Se o key pair já existe, tentar usá-lo (pode ser de uma reconciliação anterior)
			if strings.Contains(err.Error(), "InvalidKeyPair.Duplicate") {
				logger.Info("Key pair already exists, will use it", "keyPairName", keyPairName)
				keyName = keyPairName
				keyPairGenerated = true
			} else {
				return fmt.Errorf("failed to create key pair: %w", err)
			}
		} else {
			keyName = keyPairName
			keyPairGenerated = true

			// Criar Secret no Kubernetes com a chave privada
			privateKey := aws.ToString(createKeyOutput.KeyMaterial)
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: stack.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":       "computestack-ssh-key",
						"app.kubernetes.io/instance":   stack.Name,
						"app.kubernetes.io/managed-by": "infra-operator",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: stack.APIVersion,
							Kind:       stack.Kind,
							Name:       stack.Name,
							UID:        stack.UID,
						},
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"private-key":     []byte(privateKey),
					"private-key.pem": []byte(privateKey),
				},
				StringData: map[string]string{
					"key-pair-name": keyPairName,
					"ssh-user":      "ec2-user",
				},
			}

			// Criar ou atualizar o Secret
			existingSecret := &corev1.Secret{}
			err = r.Get(ctx, client.ObjectKey{Name: secretName, Namespace: stack.Namespace}, existingSecret)
			if err != nil {
				if errors.IsNotFound(err) {
					if err := r.Create(ctx, secret); err != nil {
						logger.Error(err, "Failed to create SSH key secret")
						return fmt.Errorf("failed to create SSH key secret: %w", err)
					}
					logger.Info("Created SSH key secret", "secretName", secretName)
				} else {
					return fmt.Errorf("failed to check existing secret: %w", err)
				}
			} else {
				// Secret já existe, atualizar
				existingSecret.Data = secret.Data
				existingSecret.StringData = secret.StringData
				if err := r.Update(ctx, existingSecret); err != nil {
					logger.Error(err, "Failed to update SSH key secret")
				}
			}
		}
	}

	// Obter userData (do spec ou de um Secret)
	userData := ""
	if bastionConfig.UserData != "" {
		userData = bastionConfig.UserData
	} else if bastionConfig.UserDataSecretRef != nil {
		// Buscar userData de um Secret
		secretNamespace := bastionConfig.UserDataSecretRef.Namespace
		if secretNamespace == "" {
			secretNamespace = stack.Namespace
		}
		secretKey := bastionConfig.UserDataSecretRef.Key
		if secretKey == "" {
			secretKey = "userData"
		}

		userDataSecret := &corev1.Secret{}
		err := r.Get(ctx, client.ObjectKey{
			Name:      bastionConfig.UserDataSecretRef.Name,
			Namespace: secretNamespace,
		}, userDataSecret)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to get userData secret: %w", err)
			}
			logger.Info("UserData secret not found, proceeding without userData",
				"secretName", bastionConfig.UserDataSecretRef.Name)
		} else {
			if data, ok := userDataSecret.Data[secretKey]; ok {
				userData = string(data)
			}
		}
	}

	hasUserData := userData != ""
	logger.Info("Creating Bastion Instance",
		"name", instanceName,
		"type", instanceType,
		"imageID", imageID,
		"subnetID", subnetID,
		"keyName", keyName,
		"keyPairGenerated", keyPairGenerated,
		"hasUserData", hasUserData)

	// When using NetworkInterfaces, we cannot specify SubnetId and SecurityGroupIds at the top level
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(imageID),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		KeyName:      aws.String(keyName), // Sempre terá um keyName agora
		NetworkInterfaces: []types.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int32(0),
				SubnetId:                 aws.String(subnetID),
				Groups:                   securityGroupIDs,
				AssociatePublicIpAddress: aws.Bool(associatePublicIP),
				DeleteOnTermination:      aws.Bool(true),
			},
		},
		BlockDeviceMappings: []types.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/xvda"),
				Ebs: &types.EbsBlockDevice{
					VolumeSize:          aws.Int32(rootVolumeSize),
					VolumeType:          types.VolumeTypeGp3,
					DeleteOnTermination: aws.Bool(true),
				},
			},
		},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         r.buildTags(stack, instanceName),
			},
		},
	}

	// Adicionar userData se especificado (codifica automaticamente em base64)
	if hasUserData {
		encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))
		runInput.UserData = aws.String(encodedUserData)
	}

	runOutput, err := ec2Client.RunInstances(ctx, runInput)
	if err != nil {
		return fmt.Errorf("failed to create bastion instance: %w", err)
	}

	if len(runOutput.Instances) == 0 {
		return fmt.Errorf("no instance created")
	}

	instance := runOutput.Instances[0]
	instanceID := aws.ToString(instance.InstanceId)

	// Inicializar status do bastion
	stack.Status.BastionInstance = &infrav1alpha1.BastionInstanceStatusInfo{
		ID:               instanceID,
		Name:             instanceName,
		State:            string(instance.State.Name),
		InstanceType:     instanceType,
		AvailabilityZone: aws.ToString(instance.Placement.AvailabilityZone),
		KeyPairName:      keyName,
		KeyPairGenerated: keyPairGenerated,
	}

	// Se o key pair foi gerado, adicionar referência ao Secret
	if keyPairGenerated && secretName != "" {
		stack.Status.BastionInstance.SSHKeySecretName = secretName
	}

	if instance.PrivateIpAddress != nil {
		stack.Status.BastionInstance.PrivateIP = aws.ToString(instance.PrivateIpAddress)
	}

	logger.Info("Bastion Instance created",
		"id", instanceID,
		"state", instance.State.Name,
		"keyPairName", keyName,
		"keyPairGenerated", keyPairGenerated,
		"secretName", secretName)
	return nil
}

// checkBastionReady verifica se a instância bastion está running
func (r *ComputeStackReconciler) checkBastionReady(ctx context.Context, ec2Client *ec2.Client, stack *infrav1alpha1.ComputeStack) (bool, error) {
	if stack.Status.BastionInstance == nil || stack.Status.BastionInstance.ID == "" {
		return false, nil
	}

	output, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{stack.Status.BastionInstance.ID},
	})
	if err != nil {
		return false, err
	}

	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return false, fmt.Errorf("bastion instance %s not found", stack.Status.BastionInstance.ID)
	}

	instance := output.Reservations[0].Instances[0]
	stack.Status.BastionInstance.State = string(instance.State.Name)

	if instance.PrivateIpAddress != nil {
		stack.Status.BastionInstance.PrivateIP = aws.ToString(instance.PrivateIpAddress)
	}

	if instance.PublicIpAddress != nil {
		stack.Status.BastionInstance.PublicIP = aws.ToString(instance.PublicIpAddress)
	}

	if instance.Placement != nil {
		stack.Status.BastionInstance.AvailabilityZone = aws.ToString(instance.Placement.AvailabilityZone)
	}

	// Gerar comando SSH
	if stack.Status.BastionInstance.PublicIP != "" {
		publicIP := stack.Status.BastionInstance.PublicIP

		if stack.Status.BastionInstance.KeyPairGenerated && stack.Status.BastionInstance.SSHKeySecretName != "" {
			// Key pair foi gerado automaticamente - mostrar como extrair do Secret
			secretName := stack.Status.BastionInstance.SSHKeySecretName
			namespace := stack.Namespace
			stack.Status.BastionInstance.SSHCommand = fmt.Sprintf(
				"kubectl get secret %s -n %s -o jsonpath='{.data.private-key}' | base64 -d > /tmp/%s.pem && chmod 600 /tmp/%s.pem && ssh -i /tmp/%s.pem ec2-user@%s",
				secretName, namespace, stack.Name, stack.Name, stack.Name, publicIP)
		} else if stack.Spec.BastionInstance != nil && stack.Spec.BastionInstance.KeyName != "" {
			// Key pair foi especificado pelo usuário
			stack.Status.BastionInstance.SSHCommand = fmt.Sprintf("ssh -i %s.pem ec2-user@%s",
				stack.Spec.BastionInstance.KeyName,
				publicIP)
		} else if stack.Status.BastionInstance.KeyPairName != "" {
			// Key pair existe mas não é do Secret (caso de reconecção)
			stack.Status.BastionInstance.SSHCommand = fmt.Sprintf("ssh -i %s.pem ec2-user@%s",
				stack.Status.BastionInstance.KeyPairName,
				publicIP)
		} else {
			stack.Status.BastionInstance.SSHCommand = fmt.Sprintf("ssh ec2-user@%s (no key configured)",
				publicIP)
		}
	}

	return instance.State.Name == types.InstanceStateNameRunning, nil
}

func (r *ComputeStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.ComputeStack{}).
		Complete(r)
}
