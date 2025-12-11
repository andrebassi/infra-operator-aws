package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	awsclients "infra-operator/pkg/clients"
)

const setupEKSFinalizerName = "setupeks.aws-infra-operator.runner.codes/finalizer"

// Fases do SetupEKS
const (
	EKSPhasePending              = "Pending"
	EKSPhaseCreatingIAMRoles     = "CreatingIAMRoles"
	EKSPhaseCreatingVPC          = "CreatingVPC"
	EKSPhaseWaitingVPC           = "WaitingVPC"
	EKSPhaseCreatingIGW          = "CreatingInternetGateway"
	EKSPhaseCreatingSubnets      = "CreatingSubnets"
	EKSPhaseWaitingSubnets       = "WaitingSubnets"
	EKSPhaseCreatingNATGateway   = "CreatingNATGateway"
	EKSPhaseWaitingNATGateway    = "WaitingNATGateway"
	EKSPhaseCreatingRouteTables  = "CreatingRouteTables"
	EKSPhaseCreatingSecGroups    = "CreatingSecurityGroups"
	EKSPhaseCreatingCluster      = "CreatingCluster"
	EKSPhaseWaitingCluster       = "WaitingCluster"
	EKSPhaseCreatingNodePools    = "CreatingNodePools"
	EKSPhaseWaitingNodePools     = "WaitingNodePools"
	EKSPhaseInstallingAddons     = "InstallingAddons"
	EKSPhaseWaitingAddons        = "WaitingAddons"
	EKSPhaseReady                = "Ready"
	EKSPhaseDeleting             = "Deleting"
	EKSPhaseFailed               = "Failed"
)

// IAM Policy Documents
const (
	eksClusterAssumeRolePolicy = `{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {"Service": "eks.amazonaws.com"},
			"Action": "sts:AssumeRole"
		}]
	}`

	eksNodeAssumeRolePolicy = `{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {"Service": "ec2.amazonaws.com"},
			"Action": "sts:AssumeRole"
		}]
	}`
)

// SetupEKSReconciler reconcilia um recurso SetupEKS
type SetupEKSReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	AWSClientFactory *awsclients.AWSClientFactory
}

// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=setupeks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=setupeks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws-infra-operator.runner.codes,resources=setupeks/finalizers,verbs=update

func (r *SetupEKSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Buscar o SetupEKS
	setup := &infrav1alpha1.SetupEKS{}
	if err := r.Get(ctx, req.NamespacedName, setup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get AWS clients
	awsConfig, _, err := r.AWSClientFactory.GetAWSConfigFromProviderRef(ctx, setup.Namespace, setup.Spec.ProviderRef)
	if err != nil {
		logger.Error(err, "Failed to get AWS configuration")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
	}

	ec2Client := ec2.NewFromConfig(awsConfig)
	eksClient := eks.NewFromConfig(awsConfig)
	iamClient := iam.NewFromConfig(awsConfig)
	elbv2Client := elasticloadbalancingv2.NewFromConfig(awsConfig)

	// Check if being deleted
	if !setup.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(setup, setupEKSFinalizerName) {
			if setup.Status.Phase != EKSPhaseDeleting {
				setup.Status.Phase = EKSPhaseDeleting
				setup.Status.Message = "Starting deletion of EKS resources..."
				setup.Status.Ready = false
				if err := r.Status().Update(ctx, setup); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
			}

			done, err := r.deleteSetupAsync(ctx, ec2Client, eksClient, iamClient, elbv2Client, setup)
			if err != nil {
				logger.Error(err, "Failed to delete SetupEKS resources (will retry)")
				setup.Status.Message = fmt.Sprintf("Deletion in progress: %s", err.Error())
				r.Status().Update(ctx, setup)
				return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}

			if !done {
				if err := r.Status().Update(ctx, setup); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}

			logger.Info("All SetupEKS resources deleted, removing finalizer")
			controllerutil.RemoveFinalizer(setup, setupEKSFinalizerName)
			if err := r.Update(ctx, setup); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Adicionar finalizer
	if !controllerutil.ContainsFinalizer(setup, setupEKSFinalizerName) {
		controllerutil.AddFinalizer(setup, setupEKSFinalizerName)
		if err := r.Update(ctx, setup); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Inicializar fase se vazia
	if setup.Status.Phase == "" {
		setup.Status.Phase = EKSPhasePending
		setup.Status.Message = "Initializing SetupEKS creation"
		if err := r.Status().Update(ctx, setup); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Se já está Ready, verificar drift periodicamente
	if setup.Status.Phase == EKSPhaseReady {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Processar próxima fase
	result, err := r.processNextPhase(ctx, ec2Client, eksClient, iamClient, setup)
	if err != nil {
		logger.Error(err, "Falha ao processar fase", "phase", setup.Status.Phase)
		setup.Status.Phase = EKSPhaseFailed
		setup.Status.Message = err.Error()
		setup.Status.Ready = false
		if updateErr := r.Status().Update(ctx, setup); updateErr != nil {
			logger.Error(updateErr, "Falha ao atualizar status")
		}
		return ctrl.Result{RequeueAfter: 2 * time.Minute}, err
	}

	// Atualizar status
	setup.Status.LastSyncTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, setup); err != nil {
		logger.Error(err, "Falha ao atualizar status do SetupEKS")
		return ctrl.Result{}, err
	}

	logger.Info("Fase processada",
		"name", setup.Name,
		"phase", setup.Status.Phase,
		"ready", setup.Status.Ready)

	return result, nil
}

// processNextPhase processa a próxima fase baseado no estado atual
func (r *SetupEKSReconciler) processNextPhase(ctx context.Context, ec2Client *ec2.Client, eksClient *eks.Client, iamClient *iam.Client, setup *infrav1alpha1.SetupEKS) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	switch setup.Status.Phase {
	case EKSPhasePending:
		setup.Status.Phase = EKSPhaseCreatingIAMRoles
		setup.Status.Message = "Creating IAM roles..."
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseCreatingIAMRoles:
		logger.Info("Step 1/10: Creating IAM Roles")
		if err := r.reconcileIAMRoles(ctx, iamClient, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create IAM roles: %w", err)
		}
		setup.Status.Phase = EKSPhaseCreatingVPC
		setup.Status.Message = "IAM roles created. Creating VPC..."
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseCreatingVPC:
		logger.Info("Step 2/10: Creating VPC")
		if err := r.reconcileVPC(ctx, ec2Client, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create VPC: %w", err)
		}
		setup.Status.Phase = EKSPhaseWaitingVPC
		setup.Status.Message = "VPC created. Waiting for VPC to become available..."
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil

	case EKSPhaseWaitingVPC:
		logger.Info("Step 2/10: Waiting for VPC")
		ready, err := r.checkVPCReady(ctx, ec2Client, setup)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check VPC status: %w", err)
		}
		if !ready {
			setup.Status.Message = "Waiting for VPC to become available..."
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		setup.Status.Phase = EKSPhaseCreatingIGW
		setup.Status.Message = "VPC available. Creating Internet Gateway..."
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseCreatingIGW:
		logger.Info("Step 3/10: Creating Internet Gateway")
		if err := r.reconcileInternetGateway(ctx, ec2Client, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Internet Gateway: %w", err)
		}
		setup.Status.Phase = EKSPhaseCreatingSubnets
		setup.Status.Message = "Internet Gateway created. Creating Subnets..."
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseCreatingSubnets:
		logger.Info("Step 4/10: Creating Subnets")
		if err := r.reconcileSubnets(ctx, ec2Client, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Subnets: %w", err)
		}
		setup.Status.Phase = EKSPhaseWaitingSubnets
		setup.Status.Message = "Subnets created. Waiting for subnets to become available..."
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil

	case EKSPhaseWaitingSubnets:
		logger.Info("Step 4/10: Waiting for Subnets")
		ready, err := r.checkSubnetsReady(ctx, ec2Client, setup)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check Subnets status: %w", err)
		}
		if !ready {
			setup.Status.Message = "Waiting for subnets to become available..."
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		// Verificar se precisa criar NAT Gateway
		if setup.Spec.NATGatewayMode != "None" && len(setup.Status.PrivateSubnets) > 0 {
			setup.Status.Phase = EKSPhaseCreatingNATGateway
			setup.Status.Message = "Subnets available. Creating NAT Gateway..."
		} else {
			setup.Status.Phase = EKSPhaseCreatingRouteTables
			setup.Status.Message = "Subnets available. Creating Route Tables..."
		}
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseCreatingNATGateway:
		logger.Info("Step 5/10: Creating NAT Gateway")
		if err := r.reconcileNATGateways(ctx, ec2Client, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create NAT Gateway: %w", err)
		}
		setup.Status.Phase = EKSPhaseWaitingNATGateway
		setup.Status.Message = "NAT Gateway created. Waiting for it to become available..."
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil

	case EKSPhaseWaitingNATGateway:
		logger.Info("Step 5/10: Waiting for NAT Gateway")
		ready, err := r.checkNATGatewaysReady(ctx, ec2Client, setup)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check NAT Gateway status: %w", err)
		}
		if !ready {
			setup.Status.Message = "Waiting for NAT Gateway to become available (this may take a few minutes)..."
			return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
		}
		setup.Status.Phase = EKSPhaseCreatingRouteTables
		setup.Status.Message = "NAT Gateway available. Creating Route Tables..."
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseCreatingRouteTables:
		logger.Info("Step 6/10: Creating Route Tables")
		if err := r.reconcileRouteTables(ctx, ec2Client, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Route Tables: %w", err)
		}
		setup.Status.Phase = EKSPhaseCreatingSecGroups
		setup.Status.Message = "Route Tables created. Creating Security Groups..."
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseCreatingSecGroups:
		logger.Info("Step 7/10: Creating Security Groups")
		if err := r.reconcileSecurityGroups(ctx, ec2Client, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Security Groups: %w", err)
		}
		setup.Status.Phase = EKSPhaseCreatingCluster
		setup.Status.Message = "Security Groups created. Creating EKS Cluster..."
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseCreatingCluster:
		logger.Info("Step 8/10: Creating EKS Cluster")
		if err := r.reconcileCluster(ctx, eksClient, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create EKS Cluster: %w", err)
		}
		setup.Status.Phase = EKSPhaseWaitingCluster
		setup.Status.Message = "EKS Cluster created. Waiting for cluster to become ACTIVE (this may take 10-15 minutes)..."
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	case EKSPhaseWaitingCluster:
		logger.Info("Step 8/10: Waiting for EKS Cluster")
		ready, err := r.checkClusterReady(ctx, eksClient, setup)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check EKS Cluster status: %w", err)
		}
		if !ready {
			setup.Status.Message = fmt.Sprintf("Waiting for EKS Cluster to become ACTIVE (current: %s)...", setup.Status.Cluster.Status)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		setup.Status.Phase = EKSPhaseCreatingNodePools
		setup.Status.Message = "EKS Cluster ACTIVE. Creating Node Pools..."
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseCreatingNodePools:
		logger.Info("Step 9/10: Creating Node Pools")
		if err := r.reconcileNodePools(ctx, eksClient, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Node Pools: %w", err)
		}
		setup.Status.Phase = EKSPhaseWaitingNodePools
		setup.Status.Message = "Node Pools created. Waiting for nodes to become ACTIVE..."
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	case EKSPhaseWaitingNodePools:
		logger.Info("Step 9/10: Waiting for Node Pools")
		ready, err := r.checkNodePoolsReady(ctx, eksClient, setup)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check Node Pools status: %w", err)
		}
		if !ready {
			setup.Status.Message = "Waiting for Node Pools to become ACTIVE..."
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		// Verificar se precisa instalar addons
		if setup.Spec.InstallDefaultAddons || len(setup.Spec.Addons) > 0 {
			setup.Status.Phase = EKSPhaseInstallingAddons
			setup.Status.Message = "Node Pools ACTIVE. Installing Add-ons..."
		} else {
			setup.Status.Phase = EKSPhaseReady
			setup.Status.Ready = true
			setup.Status.Message = fmt.Sprintf("SetupEKS completed! Kubeconfig: %s", setup.Status.KubeconfigCommand)
		}
		return ctrl.Result{Requeue: true}, nil

	case EKSPhaseInstallingAddons:
		logger.Info("Step 10/10: Installing Add-ons")
		if err := r.reconcileAddons(ctx, eksClient, setup); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to install Add-ons: %w", err)
		}
		setup.Status.Phase = EKSPhaseWaitingAddons
		setup.Status.Message = "Add-ons installation started. Waiting for add-ons to become ACTIVE..."
		return ctrl.Result{RequeueAfter: 20 * time.Second}, nil

	case EKSPhaseWaitingAddons:
		logger.Info("Step 10/10: Waiting for Add-ons")
		ready, err := r.checkAddonsReady(ctx, eksClient, setup)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check Add-ons status: %w", err)
		}
		if !ready {
			setup.Status.Message = "Waiting for Add-ons to become ACTIVE..."
			return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
		}
		setup.Status.Phase = EKSPhaseReady
		setup.Status.Ready = true
		setup.Status.Message = fmt.Sprintf("SetupEKS completed! Kubeconfig: %s", setup.Status.KubeconfigCommand)
		return ctrl.Result{}, nil

	case EKSPhaseFailed:
		setup.Status.Phase = EKSPhasePending
		setup.Status.Message = "Retrying setup..."
		return ctrl.Result{Requeue: true}, nil

	default:
		setup.Status.Phase = EKSPhasePending
		return ctrl.Result{Requeue: true}, nil
	}
}

// ===========================================================================
// IAM Roles
// ===========================================================================

func (r *SetupEKSReconciler) reconcileIAMRoles(ctx context.Context, iamClient *iam.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)
	clusterName := r.getClusterName(setup)

	// Create Cluster Role
	if setup.Status.ClusterRole == nil || setup.Status.ClusterRole.ARN == "" {
		clusterRoleName := fmt.Sprintf("%s-cluster-role", clusterName)
		logger.Info("Creating EKS Cluster IAM Role", "name", clusterRoleName)

		createRoleOutput, err := iamClient.CreateRole(ctx, &iam.CreateRoleInput{
			RoleName:                 aws.String(clusterRoleName),
			AssumeRolePolicyDocument: aws.String(eksClusterAssumeRolePolicy),
			Description:              aws.String(fmt.Sprintf("EKS Cluster role for %s", clusterName)),
			Tags:                     r.buildIAMTags(setup, clusterRoleName),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "EntityAlreadyExists") {
				return fmt.Errorf("failed to create cluster role: %w", err)
			}
			// Role already exists, get its ARN
			getRoleOutput, err := iamClient.GetRole(ctx, &iam.GetRoleInput{
				RoleName: aws.String(clusterRoleName),
			})
			if err != nil {
				return fmt.Errorf("failed to get existing cluster role: %w", err)
			}
			setup.Status.ClusterRole = &infrav1alpha1.IAMRoleStatusInfo{
				Name: clusterRoleName,
				ARN:  aws.ToString(getRoleOutput.Role.Arn),
			}
		} else {
			setup.Status.ClusterRole = &infrav1alpha1.IAMRoleStatusInfo{
				Name: clusterRoleName,
				ARN:  aws.ToString(createRoleOutput.Role.Arn),
			}
		}

		// Attach required policies
		policies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
		}
		for _, policyARN := range policies {
			_, err := iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
				RoleName:  aws.String(clusterRoleName),
				PolicyArn: aws.String(policyARN),
			})
			if err != nil && !strings.Contains(err.Error(), "already attached") {
				return fmt.Errorf("failed to attach policy %s: %w", policyARN, err)
			}
		}
		logger.Info("EKS Cluster IAM Role created", "arn", setup.Status.ClusterRole.ARN)
	}

	// Create Node Role
	if setup.Status.NodeRole == nil || setup.Status.NodeRole.ARN == "" {
		nodeRoleName := fmt.Sprintf("%s-node-role", clusterName)
		logger.Info("Creating EKS Node IAM Role", "name", nodeRoleName)

		createRoleOutput, err := iamClient.CreateRole(ctx, &iam.CreateRoleInput{
			RoleName:                 aws.String(nodeRoleName),
			AssumeRolePolicyDocument: aws.String(eksNodeAssumeRolePolicy),
			Description:              aws.String(fmt.Sprintf("EKS Node role for %s", clusterName)),
			Tags:                     r.buildIAMTags(setup, nodeRoleName),
		})
		if err != nil {
			if !strings.Contains(err.Error(), "EntityAlreadyExists") {
				return fmt.Errorf("failed to create node role: %w", err)
			}
			getRoleOutput, err := iamClient.GetRole(ctx, &iam.GetRoleInput{
				RoleName: aws.String(nodeRoleName),
			})
			if err != nil {
				return fmt.Errorf("failed to get existing node role: %w", err)
			}
			setup.Status.NodeRole = &infrav1alpha1.IAMRoleStatusInfo{
				Name: nodeRoleName,
				ARN:  aws.ToString(getRoleOutput.Role.Arn),
			}
		} else {
			setup.Status.NodeRole = &infrav1alpha1.IAMRoleStatusInfo{
				Name: nodeRoleName,
				ARN:  aws.ToString(createRoleOutput.Role.Arn),
			}
		}

		// Attach required policies for nodes
		nodePolicies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		}
		for _, policyARN := range nodePolicies {
			_, err := iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
				RoleName:  aws.String(nodeRoleName),
				PolicyArn: aws.String(policyARN),
			})
			if err != nil && !strings.Contains(err.Error(), "already attached") {
				return fmt.Errorf("failed to attach policy %s to node role: %w", policyARN, err)
			}
		}
		logger.Info("EKS Node IAM Role created", "arn", setup.Status.NodeRole.ARN)
	}

	return nil
}

// ===========================================================================
// VPC
// ===========================================================================

func (r *SetupEKSReconciler) reconcileVPC(ctx context.Context, ec2Client *ec2.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)

	if setup.Status.VPC != nil && setup.Status.VPC.ID != "" {
		return nil
	}

	// Se existingVpcID foi especificado, usar
	if setup.Spec.ExistingVpcID != "" {
		logger.Info("Using existing VPC", "vpcID", setup.Spec.ExistingVpcID)
		output, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
			VpcIds: []string{setup.Spec.ExistingVpcID},
		})
		if err != nil {
			return fmt.Errorf("existing VPC not found: %w", err)
		}
		if len(output.Vpcs) == 0 {
			return fmt.Errorf("existing VPC %s not found", setup.Spec.ExistingVpcID)
		}
		setup.Status.VPC = &infrav1alpha1.VPCStatusInfo{
			ID:    setup.Spec.ExistingVpcID,
			CIDR:  aws.ToString(output.Vpcs[0].CidrBlock),
			State: string(output.Vpcs[0].State),
		}
		return nil
	}

	// Criar VPC
	clusterName := r.getClusterName(setup)
	vpcName := fmt.Sprintf("%s-vpc", clusterName)

	createOutput, err := ec2Client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String(setup.Spec.VpcCIDR),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeVpc,
				Tags:         r.buildEC2Tags(setup, vpcName),
			},
		},
	})
	if err != nil {
		return err
	}

	vpcID := aws.ToString(createOutput.Vpc.VpcId)

	// Habilitar DNS
	_, err = ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:              aws.String(vpcID),
		EnableDnsHostnames: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
	})
	if err != nil {
		return err
	}

	_, err = ec2Client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:            aws.String(vpcID),
		EnableDnsSupport: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
	})
	if err != nil {
		return err
	}

	setup.Status.VPC = &infrav1alpha1.VPCStatusInfo{
		ID:    vpcID,
		CIDR:  setup.Spec.VpcCIDR,
		State: string(createOutput.Vpc.State),
	}

	logger.Info("VPC created", "vpcID", vpcID)
	return nil
}

func (r *SetupEKSReconciler) checkVPCReady(ctx context.Context, ec2Client *ec2.Client, setup *infrav1alpha1.SetupEKS) (bool, error) {
	if setup.Status.VPC == nil || setup.Status.VPC.ID == "" {
		return false, nil
	}

	output, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{setup.Status.VPC.ID},
	})
	if err != nil {
		return false, err
	}
	if len(output.Vpcs) == 0 {
		return false, fmt.Errorf("VPC %s not found", setup.Status.VPC.ID)
	}

	setup.Status.VPC.State = string(output.Vpcs[0].State)
	return output.Vpcs[0].State == ec2types.VpcStateAvailable, nil
}

// ===========================================================================
// Internet Gateway
// ===========================================================================

func (r *SetupEKSReconciler) reconcileInternetGateway(ctx context.Context, ec2Client *ec2.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)

	if setup.Status.InternetGateway != nil && setup.Status.InternetGateway.ID != "" {
		return nil
	}

	clusterName := r.getClusterName(setup)
	igwName := fmt.Sprintf("%s-igw", clusterName)

	createOutput, err := ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInternetGateway,
				Tags:         r.buildEC2Tags(setup, igwName),
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
		VpcId:             aws.String(setup.Status.VPC.ID),
	})
	if err != nil {
		return err
	}

	setup.Status.InternetGateway = &infrav1alpha1.IGWStatusInfo{
		ID:    igwID,
		State: "attached",
	}

	logger.Info("Internet Gateway created and attached", "igwID", igwID)
	return nil
}

// ===========================================================================
// Subnets
// ===========================================================================

func (r *SetupEKSReconciler) reconcileSubnets(ctx context.Context, ec2Client *ec2.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)

	// Se existingSubnetIDs foi especificado
	if len(setup.Spec.ExistingSubnetIDs) > 0 {
		logger.Info("Using existing subnets", "count", len(setup.Spec.ExistingSubnetIDs))
		output, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: setup.Spec.ExistingSubnetIDs,
		})
		if err != nil {
			return fmt.Errorf("existing subnets not found: %w", err)
		}

		setup.Status.PublicSubnets = nil
		setup.Status.PrivateSubnets = nil

		for _, subnet := range output.Subnets {
			subnetInfo := infrav1alpha1.SubnetStatusInfo{
				ID:               aws.ToString(subnet.SubnetId),
				CIDR:             aws.ToString(subnet.CidrBlock),
				AvailabilityZone: aws.ToString(subnet.AvailabilityZone),
				State:            string(subnet.State),
			}
			if subnet.MapPublicIpOnLaunch != nil && *subnet.MapPublicIpOnLaunch {
				subnetInfo.Type = "public"
				setup.Status.PublicSubnets = append(setup.Status.PublicSubnets, subnetInfo)
			} else {
				subnetInfo.Type = "private"
				setup.Status.PrivateSubnets = append(setup.Status.PrivateSubnets, subnetInfo)
			}
		}
		return nil
	}

	// Obter AZs
	azs := setup.Spec.AvailabilityZones
	if len(azs) == 0 {
		azOutput, err := ec2Client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
			Filters: []ec2types.Filter{
				{Name: aws.String("state"), Values: []string{"available"}},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to get availability zones: %w", err)
		}
		// Usar 2 AZs por padrão
		for i, az := range azOutput.AvailabilityZones {
			if i >= 2 {
				break
			}
			azs = append(azs, aws.ToString(az.ZoneName))
		}
	}

	clusterName := r.getClusterName(setup)

	// Calcular CIDRs se não especificados
	publicCIDRs := setup.Spec.PublicSubnetCIDRs
	privateCIDRs := setup.Spec.PrivateSubnetCIDRs

	if len(publicCIDRs) == 0 {
		publicCIDRs = r.calculateSubnetCIDRs(setup.Spec.VpcCIDR, len(azs), 0) // índices 0,1,2...
	}
	if len(privateCIDRs) == 0 {
		privateCIDRs = r.calculateSubnetCIDRs(setup.Spec.VpcCIDR, len(azs), 10) // índices 10,11,12...
	}

	// Criar subnets públicas
	for i, az := range azs {
		if i >= len(publicCIDRs) {
			break
		}
		if len(setup.Status.PublicSubnets) > i && setup.Status.PublicSubnets[i].ID != "" {
			continue
		}

		subnetName := fmt.Sprintf("%s-public-%s", clusterName, az[len(az)-2:])

		createOutput, err := ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
			VpcId:            aws.String(setup.Status.VPC.ID),
			CidrBlock:        aws.String(publicCIDRs[i]),
			AvailabilityZone: aws.String(az),
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeSubnet,
					Tags: append(r.buildEC2Tags(setup, subnetName),
						ec2types.Tag{Key: aws.String("kubernetes.io/role/elb"), Value: aws.String("1")},
						ec2types.Tag{Key: aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)), Value: aws.String("shared")},
					),
				},
			},
		})
		if err != nil {
			return err
		}

		subnetID := aws.ToString(createOutput.Subnet.SubnetId)

		// Habilitar auto-assign public IP
		_, err = ec2Client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
			SubnetId:            aws.String(subnetID),
			MapPublicIpOnLaunch: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		if err != nil {
			return err
		}

		setup.Status.PublicSubnets = append(setup.Status.PublicSubnets, infrav1alpha1.SubnetStatusInfo{
			ID:               subnetID,
			CIDR:             publicCIDRs[i],
			AvailabilityZone: az,
			Type:             "public",
			State:            string(createOutput.Subnet.State),
		})

		logger.Info("Public subnet created", "subnetID", subnetID, "az", az)
	}

	// Criar subnets privadas
	for i, az := range azs {
		if i >= len(privateCIDRs) {
			break
		}
		if len(setup.Status.PrivateSubnets) > i && setup.Status.PrivateSubnets[i].ID != "" {
			continue
		}

		subnetName := fmt.Sprintf("%s-private-%s", clusterName, az[len(az)-2:])

		createOutput, err := ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
			VpcId:            aws.String(setup.Status.VPC.ID),
			CidrBlock:        aws.String(privateCIDRs[i]),
			AvailabilityZone: aws.String(az),
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeSubnet,
					Tags: append(r.buildEC2Tags(setup, subnetName),
						ec2types.Tag{Key: aws.String("kubernetes.io/role/internal-elb"), Value: aws.String("1")},
						ec2types.Tag{Key: aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)), Value: aws.String("shared")},
					),
				},
			},
		})
		if err != nil {
			return err
		}

		setup.Status.PrivateSubnets = append(setup.Status.PrivateSubnets, infrav1alpha1.SubnetStatusInfo{
			ID:               aws.ToString(createOutput.Subnet.SubnetId),
			CIDR:             privateCIDRs[i],
			AvailabilityZone: az,
			Type:             "private",
			State:            string(createOutput.Subnet.State),
		})

		logger.Info("Private subnet created", "subnetID", aws.ToString(createOutput.Subnet.SubnetId), "az", az)
	}

	return nil
}

func (r *SetupEKSReconciler) checkSubnetsReady(ctx context.Context, ec2Client *ec2.Client, setup *infrav1alpha1.SetupEKS) (bool, error) {
	var subnetIDs []string
	for _, subnet := range setup.Status.PublicSubnets {
		if subnet.ID != "" {
			subnetIDs = append(subnetIDs, subnet.ID)
		}
	}
	for _, subnet := range setup.Status.PrivateSubnets {
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

	subnetStates := make(map[string]ec2types.SubnetState)
	for _, subnet := range output.Subnets {
		subnetStates[aws.ToString(subnet.SubnetId)] = subnet.State
	}

	allReady := true
	for i := range setup.Status.PublicSubnets {
		if state, ok := subnetStates[setup.Status.PublicSubnets[i].ID]; ok {
			setup.Status.PublicSubnets[i].State = string(state)
			if state != ec2types.SubnetStateAvailable {
				allReady = false
			}
		}
	}
	for i := range setup.Status.PrivateSubnets {
		if state, ok := subnetStates[setup.Status.PrivateSubnets[i].ID]; ok {
			setup.Status.PrivateSubnets[i].State = string(state)
			if state != ec2types.SubnetStateAvailable {
				allReady = false
			}
		}
	}

	return allReady, nil
}

// ===========================================================================
// NAT Gateways
// ===========================================================================

func (r *SetupEKSReconciler) reconcileNATGateways(ctx context.Context, ec2Client *ec2.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)

	if len(setup.Status.NATGateways) > 0 {
		return nil
	}

	clusterName := r.getClusterName(setup)

	// Determinar quantos NAT Gateways criar
	numNATs := 1
	if setup.Spec.NATGatewayMode == "HighAvailability" {
		numNATs = len(setup.Status.PublicSubnets)
	}

	for i := 0; i < numNATs; i++ {
		if i >= len(setup.Status.PublicSubnets) {
			break
		}
		publicSubnet := setup.Status.PublicSubnets[i]

		// Criar Elastic IP
		eipName := fmt.Sprintf("%s-nat-eip-%d", clusterName, i+1)
		eipOutput, err := ec2Client.AllocateAddress(ctx, &ec2.AllocateAddressInput{
			Domain: ec2types.DomainTypeVpc,
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeElasticIp,
					Tags:         r.buildEC2Tags(setup, eipName),
				},
			},
		})
		if err != nil {
			return err
		}

		// Criar NAT Gateway
		natName := fmt.Sprintf("%s-nat-%d", clusterName, i+1)
		natOutput, err := ec2Client.CreateNatGateway(ctx, &ec2.CreateNatGatewayInput{
			SubnetId:     aws.String(publicSubnet.ID),
			AllocationId: eipOutput.AllocationId,
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeNatgateway,
					Tags:         r.buildEC2Tags(setup, natName),
				},
			},
		})
		if err != nil {
			return err
		}

		setup.Status.NATGateways = append(setup.Status.NATGateways, infrav1alpha1.NATGatewayStatusInfo{
			ID:           aws.ToString(natOutput.NatGateway.NatGatewayId),
			ElasticIP:    aws.ToString(eipOutput.PublicIp),
			AllocationID: aws.ToString(eipOutput.AllocationId),
			SubnetID:     publicSubnet.ID,
			State:        string(natOutput.NatGateway.State),
		})

		logger.Info("NAT Gateway created", "natID", aws.ToString(natOutput.NatGateway.NatGatewayId))
	}

	return nil
}

func (r *SetupEKSReconciler) checkNATGatewaysReady(ctx context.Context, ec2Client *ec2.Client, setup *infrav1alpha1.SetupEKS) (bool, error) {
	if len(setup.Status.NATGateways) == 0 {
		return true, nil
	}

	var natIDs []string
	for _, nat := range setup.Status.NATGateways {
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
		if i < len(setup.Status.NATGateways) {
			setup.Status.NATGateways[i].State = state
		}
		if nat.State != ec2types.NatGatewayStateAvailable {
			allReady = false
		}
	}

	return allReady, nil
}

// ===========================================================================
// Route Tables
// ===========================================================================

func (r *SetupEKSReconciler) reconcileRouteTables(ctx context.Context, ec2Client *ec2.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)

	if len(setup.Status.RouteTables) > 0 {
		return nil
	}

	clusterName := r.getClusterName(setup)

	// Criar Route Table pública
	publicRTName := fmt.Sprintf("%s-public-rt", clusterName)
	publicRTOutput, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		VpcId: aws.String(setup.Status.VPC.ID),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeRouteTable,
				Tags:         r.buildEC2Tags(setup, publicRTName),
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
		GatewayId:            aws.String(setup.Status.InternetGateway.ID),
	})
	if err != nil {
		return err
	}

	// Associar subnets públicas
	var publicSubnetIDs []string
	for _, subnet := range setup.Status.PublicSubnets {
		_, err = ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
			RouteTableId: aws.String(publicRTID),
			SubnetId:     aws.String(subnet.ID),
		})
		if err != nil {
			return err
		}
		publicSubnetIDs = append(publicSubnetIDs, subnet.ID)
	}

	setup.Status.RouteTables = append(setup.Status.RouteTables, infrav1alpha1.RouteTableStatusInfo{
		ID:                publicRTID,
		Type:              "public",
		AssociatedSubnets: publicSubnetIDs,
	})

	logger.Info("Public Route Table created", "rtID", publicRTID)

	// Criar Route Tables privadas
	if len(setup.Status.PrivateSubnets) > 0 && len(setup.Status.NATGateways) > 0 {
		if setup.Spec.NATGatewayMode == "HighAvailability" {
			// Uma route table por AZ
			for i, nat := range setup.Status.NATGateways {
				if i >= len(setup.Status.PrivateSubnets) {
					break
				}

				privateRTName := fmt.Sprintf("%s-private-rt-%d", clusterName, i+1)
				privateRTOutput, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
					VpcId: aws.String(setup.Status.VPC.ID),
					TagSpecifications: []ec2types.TagSpecification{
						{
							ResourceType: ec2types.ResourceTypeRouteTable,
							Tags:         r.buildEC2Tags(setup, privateRTName),
						},
					},
				})
				if err != nil {
					return err
				}

				privateRTID := aws.ToString(privateRTOutput.RouteTable.RouteTableId)

				_, err = ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
					RouteTableId:         aws.String(privateRTID),
					DestinationCidrBlock: aws.String("0.0.0.0/0"),
					NatGatewayId:         aws.String(nat.ID),
				})
				if err != nil {
					return err
				}

				subnet := setup.Status.PrivateSubnets[i]
				_, err = ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
					RouteTableId: aws.String(privateRTID),
					SubnetId:     aws.String(subnet.ID),
				})
				if err != nil {
					return err
				}

				setup.Status.RouteTables = append(setup.Status.RouteTables, infrav1alpha1.RouteTableStatusInfo{
					ID:                privateRTID,
					Type:              "private",
					AssociatedSubnets: []string{subnet.ID},
				})

				logger.Info("Private Route Table created", "rtID", privateRTID, "nat", nat.ID)
			}
		} else {
			// Uma route table compartilhada
			privateRTName := fmt.Sprintf("%s-private-rt", clusterName)
			privateRTOutput, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
				VpcId: aws.String(setup.Status.VPC.ID),
				TagSpecifications: []ec2types.TagSpecification{
					{
						ResourceType: ec2types.ResourceTypeRouteTable,
						Tags:         r.buildEC2Tags(setup, privateRTName),
					},
				},
			})
			if err != nil {
				return err
			}

			privateRTID := aws.ToString(privateRTOutput.RouteTable.RouteTableId)

			_, err = ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
				RouteTableId:         aws.String(privateRTID),
				DestinationCidrBlock: aws.String("0.0.0.0/0"),
				NatGatewayId:         aws.String(setup.Status.NATGateways[0].ID),
			})
			if err != nil {
				return err
			}

			var privateSubnetIDs []string
			for _, subnet := range setup.Status.PrivateSubnets {
				_, err = ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
					RouteTableId: aws.String(privateRTID),
					SubnetId:     aws.String(subnet.ID),
				})
				if err != nil {
					return err
				}
				privateSubnetIDs = append(privateSubnetIDs, subnet.ID)
			}

			setup.Status.RouteTables = append(setup.Status.RouteTables, infrav1alpha1.RouteTableStatusInfo{
				ID:                privateRTID,
				Type:              "private",
				AssociatedSubnets: privateSubnetIDs,
			})

			logger.Info("Private Route Table created (shared)", "rtID", privateRTID)
		}
	}

	return nil
}

// ===========================================================================
// Security Groups
// ===========================================================================

func (r *SetupEKSReconciler) reconcileSecurityGroups(ctx context.Context, ec2Client *ec2.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)

	clusterName := r.getClusterName(setup)

	// Cluster Security Group (adicional)
	if setup.Status.ClusterSecurityGroup == nil || setup.Status.ClusterSecurityGroup.ID == "" {
		sgName := fmt.Sprintf("%s-cluster-sg", clusterName)

		createOutput, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(sgName),
			Description: aws.String(fmt.Sprintf("EKS cluster security group for %s", clusterName)),
			VpcId:       aws.String(setup.Status.VPC.ID),
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeSecurityGroup,
					Tags:         r.buildEC2Tags(setup, sgName),
				},
			},
		})
		if err != nil {
			return err
		}

		setup.Status.ClusterSecurityGroup = &infrav1alpha1.SecurityGroupStatusInfo{
			ID:   aws.ToString(createOutput.GroupId),
			Name: sgName,
		}

		logger.Info("Cluster Security Group created", "sgID", setup.Status.ClusterSecurityGroup.ID)
	}

	// Node Security Group
	if setup.Status.NodeSecurityGroup == nil || setup.Status.NodeSecurityGroup.ID == "" {
		sgName := fmt.Sprintf("%s-node-sg", clusterName)

		createOutput, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(sgName),
			Description: aws.String(fmt.Sprintf("EKS node security group for %s", clusterName)),
			VpcId:       aws.String(setup.Status.VPC.ID),
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeSecurityGroup,
					Tags: append(r.buildEC2Tags(setup, sgName),
						ec2types.Tag{Key: aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)), Value: aws.String("owned")},
					),
				},
			},
		})
		if err != nil {
			return err
		}

		nodeSGID := aws.ToString(createOutput.GroupId)

		// Regra: nós podem se comunicar entre si
		_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(nodeSGID),
			IpPermissions: []ec2types.IpPermission{
				{
					IpProtocol: aws.String("-1"),
					UserIdGroupPairs: []ec2types.UserIdGroupPair{
						{GroupId: aws.String(nodeSGID), Description: aws.String("Node to node communication")},
					},
				},
			},
		})
		if err != nil && !strings.Contains(err.Error(), "Duplicate") {
			return err
		}

		// Regra: cluster pode se comunicar com os nós
		_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(nodeSGID),
			IpPermissions: []ec2types.IpPermission{
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(1025),
					ToPort:     aws.Int32(65535),
					UserIdGroupPairs: []ec2types.UserIdGroupPair{
						{GroupId: aws.String(setup.Status.ClusterSecurityGroup.ID), Description: aws.String("Cluster to node communication")},
					},
				},
				{
					IpProtocol: aws.String("tcp"),
					FromPort:   aws.Int32(443),
					ToPort:     aws.Int32(443),
					UserIdGroupPairs: []ec2types.UserIdGroupPair{
						{GroupId: aws.String(setup.Status.ClusterSecurityGroup.ID), Description: aws.String("Cluster API to kubelet")},
					},
				},
			},
		})
		if err != nil && !strings.Contains(err.Error(), "Duplicate") {
			return err
		}

		setup.Status.NodeSecurityGroup = &infrav1alpha1.SecurityGroupStatusInfo{
			ID:   nodeSGID,
			Name: sgName,
		}

		logger.Info("Node Security Group created", "sgID", nodeSGID)
	}

	return nil
}

// ===========================================================================
// EKS Cluster
// ===========================================================================

func (r *SetupEKSReconciler) reconcileCluster(ctx context.Context, eksClient *eks.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)

	if setup.Status.Cluster != nil && setup.Status.Cluster.ARN != "" {
		return nil
	}

	clusterName := r.getClusterName(setup)

	// Coletar todas as subnets
	var subnetIDs []string
	for _, subnet := range setup.Status.PublicSubnets {
		subnetIDs = append(subnetIDs, subnet.ID)
	}
	for _, subnet := range setup.Status.PrivateSubnets {
		subnetIDs = append(subnetIDs, subnet.ID)
	}

	// Security groups
	securityGroupIDs := []string{}
	if setup.Status.ClusterSecurityGroup != nil {
		securityGroupIDs = append(securityGroupIDs, setup.Status.ClusterSecurityGroup.ID)
	}

	// Configuração de endpoint
	endpointPublicAccess := true
	endpointPrivateAccess := true
	var publicAccessCidrs []string

	if setup.Spec.EndpointAccess != nil {
		endpointPublicAccess = setup.Spec.EndpointAccess.PublicAccess
		endpointPrivateAccess = setup.Spec.EndpointAccess.PrivateAccess
		publicAccessCidrs = setup.Spec.EndpointAccess.PublicAccessCIDRs
	}

	if len(publicAccessCidrs) == 0 {
		publicAccessCidrs = []string{"0.0.0.0/0"}
	}

	// Logging
	var enabledLogging []ekstypes.LogType
	if setup.Spec.ClusterLogging != nil {
		if setup.Spec.ClusterLogging.APIServer {
			enabledLogging = append(enabledLogging, ekstypes.LogTypeApi)
		}
		if setup.Spec.ClusterLogging.Audit {
			enabledLogging = append(enabledLogging, ekstypes.LogTypeAudit)
		}
		if setup.Spec.ClusterLogging.Authenticator {
			enabledLogging = append(enabledLogging, ekstypes.LogTypeAuthenticator)
		}
		if setup.Spec.ClusterLogging.ControllerManager {
			enabledLogging = append(enabledLogging, ekstypes.LogTypeControllerManager)
		}
		if setup.Spec.ClusterLogging.Scheduler {
			enabledLogging = append(enabledLogging, ekstypes.LogTypeScheduler)
		}
	}

	version := setup.Spec.KubernetesVersion
	if version == "" {
		version = "1.29"
	}

	createInput := &eks.CreateClusterInput{
		Name:    aws.String(clusterName),
		Version: aws.String(version),
		RoleArn: aws.String(setup.Status.ClusterRole.ARN),
		ResourcesVpcConfig: &ekstypes.VpcConfigRequest{
			SubnetIds:             subnetIDs,
			SecurityGroupIds:      securityGroupIDs,
			EndpointPublicAccess:  aws.Bool(endpointPublicAccess),
			EndpointPrivateAccess: aws.Bool(endpointPrivateAccess),
			PublicAccessCidrs:     publicAccessCidrs,
		},
		Tags: r.buildStringTags(setup),
	}

	// Logging
	if len(enabledLogging) > 0 {
		createInput.Logging = &ekstypes.Logging{
			ClusterLogging: []ekstypes.LogSetup{
				{
					Enabled: aws.Bool(true),
					Types:   enabledLogging,
				},
			},
		}
	}

	logger.Info("Creating EKS Cluster", "name", clusterName, "version", version)

	createOutput, err := eksClient.CreateCluster(ctx, createInput)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceInUseException") {
			// Cluster já existe, buscar informações
			describeOutput, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{
				Name: aws.String(clusterName),
			})
			if err != nil {
				return err
			}
			setup.Status.Cluster = &infrav1alpha1.EKSClusterStatusInfo{
				Name:                 clusterName,
				ARN:                  aws.ToString(describeOutput.Cluster.Arn),
				Endpoint:             aws.ToString(describeOutput.Cluster.Endpoint),
				CertificateAuthority: aws.ToString(describeOutput.Cluster.CertificateAuthority.Data),
				Version:              aws.ToString(describeOutput.Cluster.Version),
				PlatformVersion:      aws.ToString(describeOutput.Cluster.PlatformVersion),
				Status:               string(describeOutput.Cluster.Status),
			}
			return nil
		}
		return err
	}

	setup.Status.Cluster = &infrav1alpha1.EKSClusterStatusInfo{
		Name:    clusterName,
		ARN:     aws.ToString(createOutput.Cluster.Arn),
		Version: aws.ToString(createOutput.Cluster.Version),
		Status:  string(createOutput.Cluster.Status),
	}

	// Gerar comando kubeconfig
	setup.Status.KubeconfigCommand = fmt.Sprintf("aws eks update-kubeconfig --name %s --region %s", clusterName, r.getRegion(ctx))

	logger.Info("EKS Cluster creation initiated", "arn", setup.Status.Cluster.ARN)
	return nil
}

func (r *SetupEKSReconciler) checkClusterReady(ctx context.Context, eksClient *eks.Client, setup *infrav1alpha1.SetupEKS) (bool, error) {
	if setup.Status.Cluster == nil {
		return false, nil
	}

	clusterName := r.getClusterName(setup)

	output, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		return false, err
	}

	setup.Status.Cluster.Status = string(output.Cluster.Status)
	setup.Status.Cluster.Endpoint = aws.ToString(output.Cluster.Endpoint)
	setup.Status.Cluster.PlatformVersion = aws.ToString(output.Cluster.PlatformVersion)

	if output.Cluster.CertificateAuthority != nil {
		setup.Status.Cluster.CertificateAuthority = aws.ToString(output.Cluster.CertificateAuthority.Data)
	}

	if output.Cluster.Identity != nil && output.Cluster.Identity.Oidc != nil {
		setup.Status.OIDCIssuerURL = aws.ToString(output.Cluster.Identity.Oidc.Issuer)
	}

	return output.Cluster.Status == ekstypes.ClusterStatusActive, nil
}

// ===========================================================================
// Node Pools
// ===========================================================================

func (r *SetupEKSReconciler) reconcileNodePools(ctx context.Context, eksClient *eks.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)

	clusterName := r.getClusterName(setup)

	for _, poolSpec := range setup.Spec.NodePools {
		// Verificar se já existe no status
		exists := false
		for _, existing := range setup.Status.NodePools {
			if existing.Name == poolSpec.Name {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		nodeGroupName := fmt.Sprintf("%s-%s", clusterName, poolSpec.Name)

		// Selecionar subnets
		var subnetIDs []string
		switch poolSpec.SubnetSelector {
		case "public":
			for _, subnet := range setup.Status.PublicSubnets {
				subnetIDs = append(subnetIDs, subnet.ID)
			}
		case "all":
			for _, subnet := range setup.Status.PublicSubnets {
				subnetIDs = append(subnetIDs, subnet.ID)
			}
			for _, subnet := range setup.Status.PrivateSubnets {
				subnetIDs = append(subnetIDs, subnet.ID)
			}
		default: // private
			for _, subnet := range setup.Status.PrivateSubnets {
				subnetIDs = append(subnetIDs, subnet.ID)
			}
			// Se não há private, usar public
			if len(subnetIDs) == 0 {
				for _, subnet := range setup.Status.PublicSubnets {
					subnetIDs = append(subnetIDs, subnet.ID)
				}
			}
		}

		// Scaling config
		minSize := int32(1)
		maxSize := int32(3)
		desiredSize := int32(2)
		if poolSpec.ScalingConfig != nil {
			if poolSpec.ScalingConfig.MinSize > 0 {
				minSize = poolSpec.ScalingConfig.MinSize
			}
			if poolSpec.ScalingConfig.MaxSize > 0 {
				maxSize = poolSpec.ScalingConfig.MaxSize
			}
			if poolSpec.ScalingConfig.DesiredSize > 0 {
				desiredSize = poolSpec.ScalingConfig.DesiredSize
			}
		}

		// Capacity type
		capacityType := ekstypes.CapacityTypesOnDemand
		if poolSpec.CapacityType == "SPOT" {
			capacityType = ekstypes.CapacityTypesSpot
		}

		// AMI type
		amiType := ekstypes.AMITypesAl2X8664
		switch poolSpec.AMIType {
		case "AL2_ARM_64":
			amiType = ekstypes.AMITypesAl2Arm64
		case "AL2_x86_64_GPU":
			amiType = ekstypes.AMITypesAl2X8664Gpu
		case "BOTTLEROCKET_x86_64":
			amiType = ekstypes.AMITypesBottlerocketX8664
		case "BOTTLEROCKET_ARM_64":
			amiType = ekstypes.AMITypesBottlerocketArm64
		}

		// Disk size
		diskSize := int32(50)
		if poolSpec.DiskSize > 0 {
			diskSize = poolSpec.DiskSize
		}

		// Labels
		labels := make(map[string]string)
		if setup.Spec.DefaultNodePool != nil && setup.Spec.DefaultNodePool.Labels != nil {
			for k, v := range setup.Spec.DefaultNodePool.Labels {
				labels[k] = v
			}
		}
		if poolSpec.Labels != nil {
			for k, v := range poolSpec.Labels {
				labels[k] = v
			}
		}

		// Taints
		var taints []ekstypes.Taint
		for _, t := range poolSpec.Taints {
			effect := ekstypes.TaintEffectNoSchedule
			switch t.Effect {
			case "NO_EXECUTE":
				effect = ekstypes.TaintEffectNoExecute
			case "PREFER_NO_SCHEDULE":
				effect = ekstypes.TaintEffectPreferNoSchedule
			}
			taints = append(taints, ekstypes.Taint{
				Key:    aws.String(t.Key),
				Value:  aws.String(t.Value),
				Effect: effect,
			})
		}

		// Tags
		tags := r.buildStringTags(setup)
		if poolSpec.Tags != nil {
			for k, v := range poolSpec.Tags {
				tags[k] = v
			}
		}

		createInput := &eks.CreateNodegroupInput{
			ClusterName:   aws.String(clusterName),
			NodegroupName: aws.String(nodeGroupName),
			NodeRole:      aws.String(setup.Status.NodeRole.ARN),
			Subnets:       subnetIDs,
			InstanceTypes: poolSpec.InstanceTypes,
			ScalingConfig: &ekstypes.NodegroupScalingConfig{
				MinSize:     aws.Int32(minSize),
				MaxSize:     aws.Int32(maxSize),
				DesiredSize: aws.Int32(desiredSize),
			},
			CapacityType: capacityType,
			AmiType:      amiType,
			DiskSize:     aws.Int32(diskSize),
			Labels:       labels,
			Tags:         tags,
		}

		if len(taints) > 0 {
			createInput.Taints = taints
		}

		// Update config
		if poolSpec.UpdateConfig != nil {
			createInput.UpdateConfig = &ekstypes.NodegroupUpdateConfig{}
			if poolSpec.UpdateConfig.MaxUnavailable > 0 {
				createInput.UpdateConfig.MaxUnavailable = aws.Int32(poolSpec.UpdateConfig.MaxUnavailable)
			}
			if poolSpec.UpdateConfig.MaxUnavailablePercentage > 0 {
				createInput.UpdateConfig.MaxUnavailablePercentage = aws.Int32(poolSpec.UpdateConfig.MaxUnavailablePercentage)
			}
		}

		logger.Info("Creating Node Pool", "name", nodeGroupName)

		createOutput, err := eksClient.CreateNodegroup(ctx, createInput)
		if err != nil {
			if strings.Contains(err.Error(), "ResourceInUseException") {
				// Node group já existe
				describeOutput, err := eksClient.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
					ClusterName:   aws.String(clusterName),
					NodegroupName: aws.String(nodeGroupName),
				})
				if err != nil {
					return err
				}
				setup.Status.NodePools = append(setup.Status.NodePools, infrav1alpha1.NodePoolStatusInfo{
					Name:          poolSpec.Name,
					ARN:           aws.ToString(describeOutput.Nodegroup.NodegroupArn),
					Status:        string(describeOutput.Nodegroup.Status),
					CapacityType:  string(describeOutput.Nodegroup.CapacityType),
					InstanceTypes: describeOutput.Nodegroup.InstanceTypes,
					DesiredSize:   aws.ToInt32(describeOutput.Nodegroup.ScalingConfig.DesiredSize),
					MinSize:       aws.ToInt32(describeOutput.Nodegroup.ScalingConfig.MinSize),
					MaxSize:       aws.ToInt32(describeOutput.Nodegroup.ScalingConfig.MaxSize),
					Subnets:       describeOutput.Nodegroup.Subnets,
				})
				continue
			}
			return fmt.Errorf("failed to create node group %s: %w", poolSpec.Name, err)
		}

		setup.Status.NodePools = append(setup.Status.NodePools, infrav1alpha1.NodePoolStatusInfo{
			Name:          poolSpec.Name,
			ARN:           aws.ToString(createOutput.Nodegroup.NodegroupArn),
			Status:        string(createOutput.Nodegroup.Status),
			CapacityType:  string(createOutput.Nodegroup.CapacityType),
			InstanceTypes: poolSpec.InstanceTypes,
			DesiredSize:   desiredSize,
			MinSize:       minSize,
			MaxSize:       maxSize,
			Subnets:       subnetIDs,
		})

		logger.Info("Node Pool created", "name", nodeGroupName, "arn", createOutput.Nodegroup.NodegroupArn)
	}

	return nil
}

func (r *SetupEKSReconciler) checkNodePoolsReady(ctx context.Context, eksClient *eks.Client, setup *infrav1alpha1.SetupEKS) (bool, error) {
	clusterName := r.getClusterName(setup)

	allReady := true
	for i, pool := range setup.Status.NodePools {
		nodeGroupName := fmt.Sprintf("%s-%s", clusterName, pool.Name)

		output, err := eksClient.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
			ClusterName:   aws.String(clusterName),
			NodegroupName: aws.String(nodeGroupName),
		})
		if err != nil {
			return false, err
		}

		setup.Status.NodePools[i].Status = string(output.Nodegroup.Status)

		if output.Nodegroup.Status != ekstypes.NodegroupStatusActive {
			allReady = false
		}
	}

	return allReady, nil
}

// ===========================================================================
// Add-ons
// ===========================================================================

func (r *SetupEKSReconciler) reconcileAddons(ctx context.Context, eksClient *eks.Client, setup *infrav1alpha1.SetupEKS) error {
	logger := log.FromContext(ctx)

	clusterName := r.getClusterName(setup)

	// Add-ons padrão
	defaultAddons := []infrav1alpha1.EKSAddonConfig{}
	if setup.Spec.InstallDefaultAddons {
		defaultAddons = []infrav1alpha1.EKSAddonConfig{
			{Name: "vpc-cni", ResolveConflicts: "OVERWRITE"},
			{Name: "coredns", ResolveConflicts: "OVERWRITE"},
			{Name: "kube-proxy", ResolveConflicts: "OVERWRITE"},
		}
	}

	// Combinar add-ons padrão com os especificados
	allAddons := append(defaultAddons, setup.Spec.Addons...)

	for _, addon := range allAddons {
		// Verificar se já existe no status
		exists := false
		for _, existing := range setup.Status.Addons {
			if existing.Name == addon.Name {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		createInput := &eks.CreateAddonInput{
			ClusterName: aws.String(clusterName),
			AddonName:   aws.String(addon.Name),
		}

		if addon.Version != "" {
			createInput.AddonVersion = aws.String(addon.Version)
		}

		if addon.ServiceAccountRoleARN != "" {
			createInput.ServiceAccountRoleArn = aws.String(addon.ServiceAccountRoleARN)
		}

		if addon.ConfigurationValues != "" {
			createInput.ConfigurationValues = aws.String(addon.ConfigurationValues)
		}

		resolveConflicts := ekstypes.ResolveConflictsOverwrite
		switch addon.ResolveConflicts {
		case "NONE":
			resolveConflicts = ekstypes.ResolveConflictsNone
		case "PRESERVE":
			resolveConflicts = ekstypes.ResolveConflictsPreserve
		}
		createInput.ResolveConflicts = resolveConflicts

		logger.Info("Creating EKS Add-on", "name", addon.Name)

		createOutput, err := eksClient.CreateAddon(ctx, createInput)
		if err != nil {
			if strings.Contains(err.Error(), "ResourceInUseException") || strings.Contains(err.Error(), "ResourceNotFoundException") {
				// Addon já existe ou cluster não está pronto
				describeOutput, err := eksClient.DescribeAddon(ctx, &eks.DescribeAddonInput{
					ClusterName: aws.String(clusterName),
					AddonName:   aws.String(addon.Name),
				})
				if err == nil {
					setup.Status.Addons = append(setup.Status.Addons, infrav1alpha1.AddonStatusInfo{
						Name:    addon.Name,
						Version: aws.ToString(describeOutput.Addon.AddonVersion),
						Status:  string(describeOutput.Addon.Status),
					})
				}
				continue
			}
			logger.Error(err, "Failed to create addon", "name", addon.Name)
			continue // Continue with other addons
		}

		setup.Status.Addons = append(setup.Status.Addons, infrav1alpha1.AddonStatusInfo{
			Name:    addon.Name,
			Version: aws.ToString(createOutput.Addon.AddonVersion),
			Status:  string(createOutput.Addon.Status),
		})

		logger.Info("Add-on created", "name", addon.Name, "version", aws.ToString(createOutput.Addon.AddonVersion))
	}

	return nil
}

func (r *SetupEKSReconciler) checkAddonsReady(ctx context.Context, eksClient *eks.Client, setup *infrav1alpha1.SetupEKS) (bool, error) {
	clusterName := r.getClusterName(setup)

	allReady := true
	for i, addon := range setup.Status.Addons {
		output, err := eksClient.DescribeAddon(ctx, &eks.DescribeAddonInput{
			ClusterName: aws.String(clusterName),
			AddonName:   aws.String(addon.Name),
		})
		if err != nil {
			continue // Addon may not be ready yet
		}

		setup.Status.Addons[i].Status = string(output.Addon.Status)
		setup.Status.Addons[i].Version = aws.ToString(output.Addon.AddonVersion)

		if output.Addon.Status != ekstypes.AddonStatusActive {
			allReady = false
		}
	}

	return allReady, nil
}

// ===========================================================================
// Deletion
// ===========================================================================

func (r *SetupEKSReconciler) deleteSetupAsync(ctx context.Context, ec2Client *ec2.Client, eksClient *eks.Client, iamClient *iam.Client, elbv2Client *elasticloadbalancingv2.Client, setup *infrav1alpha1.SetupEKS) (bool, error) {
	logger := log.FromContext(ctx)

	if setup.Spec.DeletionPolicy == "Retain" {
		logger.Info("DeletionPolicy is Retain, keeping resources in AWS")
		return true, nil
	}

	clusterName := r.getClusterName(setup)

	// Step 1: Delete Node Groups
	if len(setup.Status.NodePools) > 0 {
		for _, pool := range setup.Status.NodePools {
			nodeGroupName := fmt.Sprintf("%s-%s", clusterName, pool.Name)

			logger.Info("Deleting Node Group", "name", nodeGroupName)
			_, err := eksClient.DeleteNodegroup(ctx, &eks.DeleteNodegroupInput{
				ClusterName:   aws.String(clusterName),
				NodegroupName: aws.String(nodeGroupName),
			})
			if err != nil && !strings.Contains(err.Error(), "ResourceNotFoundException") {
				setup.Status.Message = fmt.Sprintf("Deleting node group %s...", nodeGroupName)
				return false, nil
			}
		}

		// Check if all node groups are deleted
		listOutput, err := eksClient.ListNodegroups(ctx, &eks.ListNodegroupsInput{
			ClusterName: aws.String(clusterName),
		})
		if err == nil && len(listOutput.Nodegroups) > 0 {
			setup.Status.Message = fmt.Sprintf("Waiting for %d node groups to be deleted...", len(listOutput.Nodegroups))
			return false, nil
		}

		setup.Status.NodePools = nil
	}

	// Step 2: Delete Add-ons
	if len(setup.Status.Addons) > 0 {
		for _, addon := range setup.Status.Addons {
			logger.Info("Deleting Add-on", "name", addon.Name)
			_, err := eksClient.DeleteAddon(ctx, &eks.DeleteAddonInput{
				ClusterName: aws.String(clusterName),
				AddonName:   aws.String(addon.Name),
			})
			if err != nil && !strings.Contains(err.Error(), "ResourceNotFoundException") {
				logger.Error(err, "Failed to delete addon", "name", addon.Name)
			}
		}
		setup.Status.Addons = nil
	}

	// Step 3: Delete Cluster
	if setup.Status.Cluster != nil && setup.Status.Cluster.ARN != "" {
		logger.Info("Deleting EKS Cluster", "name", clusterName)
		_, err := eksClient.DeleteCluster(ctx, &eks.DeleteClusterInput{
			Name: aws.String(clusterName),
		})
		if err != nil {
			if strings.Contains(err.Error(), "ResourceNotFoundException") {
				setup.Status.Cluster = nil
			} else {
				setup.Status.Message = fmt.Sprintf("Deleting EKS cluster %s...", clusterName)
				return false, nil
			}
		} else {
			// Wait for cluster to be deleted
			describeOutput, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{
				Name: aws.String(clusterName),
			})
			if err == nil && describeOutput.Cluster != nil {
				setup.Status.Message = fmt.Sprintf("Waiting for EKS cluster to be deleted (status: %s)...", describeOutput.Cluster.Status)
				return false, nil
			}
			setup.Status.Cluster = nil
		}
	}

	// Step 4: Delete Security Groups
	if setup.Status.NodeSecurityGroup != nil && setup.Status.NodeSecurityGroup.ID != "" {
		logger.Info("Deleting Node Security Group", "id", setup.Status.NodeSecurityGroup.ID)
		_, err := ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(setup.Status.NodeSecurityGroup.ID),
		})
		if err != nil && !isNotFoundError(err) {
			if strings.Contains(err.Error(), "DependencyViolation") {
				setup.Status.Message = "Waiting for node security group dependencies to be released..."
				return false, nil
			}
		}
		setup.Status.NodeSecurityGroup = nil
	}

	if setup.Status.ClusterSecurityGroup != nil && setup.Status.ClusterSecurityGroup.ID != "" {
		logger.Info("Deleting Cluster Security Group", "id", setup.Status.ClusterSecurityGroup.ID)
		_, err := ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(setup.Status.ClusterSecurityGroup.ID),
		})
		if err != nil && !isNotFoundError(err) {
			if strings.Contains(err.Error(), "DependencyViolation") {
				setup.Status.Message = "Waiting for cluster security group dependencies to be released..."
				return false, nil
			}
		}
		setup.Status.ClusterSecurityGroup = nil
	}

	// Step 5: Delete Route Tables (ALL route tables in VPC, not just tracked ones)
	if setup.Spec.ExistingVpcID == "" && setup.Status.VPC != nil && setup.Status.VPC.ID != "" {
		// List ALL route tables in this VPC
		descOut, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			Filters: []ec2types.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: []string{setup.Status.VPC.ID},
				},
			},
		})
		if err != nil {
			logger.Error(err, "Failed to list route tables in VPC", "vpcId", setup.Status.VPC.ID)
		} else {
			for _, rt := range descOut.RouteTables {
				rtID := aws.ToString(rt.RouteTableId)

				// Check if it's the main route table (deleted with VPC)
				isMain := false
				for _, assoc := range rt.Associations {
					if assoc.Main != nil && *assoc.Main {
						isMain = true
						break
					}
				}
				if isMain {
					logger.Info("Skipping main route table (deleted with VPC)", "id", rtID)
					continue
				}

				logger.Info("Deleting Route Table", "id", rtID)

				// Disassociate all subnet associations first
				for _, assoc := range rt.Associations {
					if assoc.RouteTableAssociationId != nil {
						logger.Info("Disassociating route table", "associationId", aws.ToString(assoc.RouteTableAssociationId))
						_, err := ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
							AssociationId: assoc.RouteTableAssociationId,
						})
						if err != nil && !isNotFoundError(err) {
							logger.Error(err, "Failed to disassociate route table", "associationId", aws.ToString(assoc.RouteTableAssociationId))
						}
					}
				}

				// Delete the route table
				_, err = ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
					RouteTableId: aws.String(rtID),
				})
				if err != nil && !isNotFoundError(err) {
					if strings.Contains(err.Error(), "DependencyViolation") {
						setup.Status.Message = fmt.Sprintf("Waiting for route table %s dependencies to be released...", rtID)
						return false, nil
					}
					logger.Error(err, "Failed to delete route table", "id", rtID)
				}
			}
		}
		setup.Status.RouteTables = nil
	}

	// Step 6: Delete NAT Gateways
	if len(setup.Spec.ExistingSubnetIDs) == 0 && len(setup.Status.NATGateways) > 0 {
		for _, nat := range setup.Status.NATGateways {
			logger.Info("Deleting NAT Gateway", "id", nat.ID)
			_, err := ec2Client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
				NatGatewayId: aws.String(nat.ID),
			})
			if err != nil && !isNotFoundError(err) {
				setup.Status.Message = fmt.Sprintf("Deleting NAT Gateway %s...", nat.ID)
				return false, nil
			}
		}

		// Check if deleted
		for _, nat := range setup.Status.NATGateways {
			descOut, err := ec2Client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
				NatGatewayIds: []string{nat.ID},
			})
			if err == nil && len(descOut.NatGateways) > 0 {
				state := descOut.NatGateways[0].State
				if state != ec2types.NatGatewayStateDeleted && state != ec2types.NatGatewayStateFailed {
					setup.Status.Message = fmt.Sprintf("Waiting for NAT Gateway %s to be deleted (state: %s)...", nat.ID, state)
					return false, nil
				}
			}

			// Release EIP
			if nat.AllocationID != "" {
				ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
					AllocationId: aws.String(nat.AllocationID),
				})
			}
		}
		setup.Status.NATGateways = nil
	}

	// Step 6.5: Delete ALL Load Balancers in VPC (ALB, NLB, CLB)
	// This is critical because Kubernetes services create LoadBalancers that are not tracked by the operator
	if setup.Spec.ExistingVpcID == "" && setup.Status.VPC != nil && setup.Status.VPC.ID != "" {
		vpcID := setup.Status.VPC.ID

		// Get all subnets in this VPC to find LoadBalancers
		allSubnetIDs := []string{}
		for _, s := range setup.Status.PublicSubnets {
			allSubnetIDs = append(allSubnetIDs, s.ID)
		}
		for _, s := range setup.Status.PrivateSubnets {
			allSubnetIDs = append(allSubnetIDs, s.ID)
		}

		// List all ELBv2 (ALB/NLB) LoadBalancers
		lbOutput, err := elbv2Client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
		if err != nil {
			logger.Error(err, "Failed to list load balancers")
		} else {
			for _, lb := range lbOutput.LoadBalancers {
				// Check if this LB is in our VPC
				if aws.ToString(lb.VpcId) == vpcID {
					lbArn := aws.ToString(lb.LoadBalancerArn)
					lbName := aws.ToString(lb.LoadBalancerName)
					logger.Info("Found LoadBalancer in VPC, deleting...", "name", lbName, "arn", lbArn, "type", lb.Type)

					// First, delete all listeners
					listenersOutput, err := elbv2Client.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{
						LoadBalancerArn: lb.LoadBalancerArn,
					})
					if err == nil {
						for _, listener := range listenersOutput.Listeners {
							logger.Info("Deleting listener", "arn", aws.ToString(listener.ListenerArn))
							elbv2Client.DeleteListener(ctx, &elasticloadbalancingv2.DeleteListenerInput{
								ListenerArn: listener.ListenerArn,
							})
						}
					}

					// Delete the LoadBalancer
					_, err = elbv2Client.DeleteLoadBalancer(ctx, &elasticloadbalancingv2.DeleteLoadBalancerInput{
						LoadBalancerArn: lb.LoadBalancerArn,
					})
					if err != nil {
						logger.Error(err, "Failed to delete load balancer", "name", lbName)
					} else {
						logger.Info("LoadBalancer deletion initiated", "name", lbName)
					}
				}
			}
		}

		// Check if any LoadBalancers still exist in VPC (wait for them to be deleted)
		lbOutput, err = elbv2Client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
		if err == nil {
			for _, lb := range lbOutput.LoadBalancers {
				if aws.ToString(lb.VpcId) == vpcID {
					// LoadBalancer still exists, wait for it
					setup.Status.Message = fmt.Sprintf("Waiting for LoadBalancer %s to be deleted...", aws.ToString(lb.LoadBalancerName))
					return false, nil
				}
			}
		}

		// Also delete Target Groups in VPC
		tgOutput, err := elbv2Client.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{})
		if err == nil {
			for _, tg := range tgOutput.TargetGroups {
				if aws.ToString(tg.VpcId) == vpcID {
					logger.Info("Deleting orphan Target Group", "name", aws.ToString(tg.TargetGroupName), "arn", aws.ToString(tg.TargetGroupArn))
					_, err := elbv2Client.DeleteTargetGroup(ctx, &elasticloadbalancingv2.DeleteTargetGroupInput{
						TargetGroupArn: tg.TargetGroupArn,
					})
					if err != nil {
						logger.Error(err, "Failed to delete target group", "name", aws.ToString(tg.TargetGroupName))
					}
				}
			}
		}
	}

	// Step 7: Delete Subnets
	if len(setup.Spec.ExistingSubnetIDs) == 0 {
		allSubnets := append(setup.Status.PublicSubnets, setup.Status.PrivateSubnets...)
		for _, subnet := range allSubnets {
			logger.Info("Deleting Subnet", "id", subnet.ID)
			_, err := ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
				SubnetId: aws.String(subnet.ID),
			})
			if err != nil && !isNotFoundError(err) {
				if strings.Contains(err.Error(), "DependencyViolation") {
					setup.Status.Message = fmt.Sprintf("Waiting for subnet %s dependencies to be released...", subnet.ID)
					return false, nil
				}
			}
		}
		setup.Status.PublicSubnets = nil
		setup.Status.PrivateSubnets = nil
	}

	// Step 8: Delete Internet Gateway
	if setup.Spec.ExistingVpcID == "" && setup.Status.InternetGateway != nil && setup.Status.InternetGateway.ID != "" {
		logger.Info("Deleting Internet Gateway", "id", setup.Status.InternetGateway.ID)

		if setup.Status.VPC != nil && setup.Status.VPC.ID != "" {
			ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
				InternetGatewayId: aws.String(setup.Status.InternetGateway.ID),
				VpcId:             aws.String(setup.Status.VPC.ID),
			})
		}

		_, err := ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(setup.Status.InternetGateway.ID),
		})
		if err != nil && !isNotFoundError(err) {
			logger.Error(err, "Failed to delete internet gateway")
		}
		setup.Status.InternetGateway = nil
	}

	// Step 9: Delete VPC
	if setup.Spec.ExistingVpcID == "" && setup.Status.VPC != nil && setup.Status.VPC.ID != "" {
		logger.Info("Deleting VPC", "id", setup.Status.VPC.ID)
		_, err := ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
			VpcId: aws.String(setup.Status.VPC.ID),
		})
		if err != nil && !isNotFoundError(err) {
			if strings.Contains(err.Error(), "DependencyViolation") {
				setup.Status.Message = fmt.Sprintf("Waiting for VPC %s dependencies to be released...", setup.Status.VPC.ID)
				return false, nil
			}
		}
		setup.Status.VPC = nil
	}

	// Step 10: Delete IAM Roles
	if setup.Status.NodeRole != nil && setup.Status.NodeRole.ARN != "" {
		roleName := setup.Status.NodeRole.Name
		logger.Info("Deleting Node IAM Role", "name", roleName)

		// Detach policies
		nodePolicies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		}
		for _, policyARN := range nodePolicies {
			iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				RoleName:  aws.String(roleName),
				PolicyArn: aws.String(policyARN),
			})
		}

		_, err := iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil && !strings.Contains(err.Error(), "NoSuchEntity") {
			logger.Error(err, "Failed to delete node role")
		}
		setup.Status.NodeRole = nil
	}

	if setup.Status.ClusterRole != nil && setup.Status.ClusterRole.ARN != "" {
		roleName := setup.Status.ClusterRole.Name
		logger.Info("Deleting Cluster IAM Role", "name", roleName)

		// Detach policies
		iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
			RoleName:  aws.String(roleName),
			PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
		})

		_, err := iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil && !strings.Contains(err.Error(), "NoSuchEntity") {
			logger.Error(err, "Failed to delete cluster role")
		}
		setup.Status.ClusterRole = nil
	}

	logger.Info("All SetupEKS resources deleted successfully")
	return true, nil
}

// ===========================================================================
// Helper Functions
// ===========================================================================

func (r *SetupEKSReconciler) getClusterName(setup *infrav1alpha1.SetupEKS) string {
	if setup.Spec.ClusterName != "" {
		return setup.Spec.ClusterName
	}
	return setup.Name
}

func (r *SetupEKSReconciler) getRegion(ctx context.Context) string {
	// Default region - should be obtained from provider
	return "us-east-1"
}

func (r *SetupEKSReconciler) buildEC2Tags(setup *infrav1alpha1.SetupEKS, name string) []ec2types.Tag {
	// Usar map para evitar duplicação de tags (case-insensitive)
	tagMap := make(map[string]string)

	// Tags padrão
	tagMap["Name"] = name
	tagMap["ManagedBy"] = "infra-operator"
	tagMap["SetupEKS"] = setup.Name

	// Tags do usuário (podem sobrescrever as padrões)
	for k, v := range setup.Spec.Tags {
		tagMap[k] = v
	}

	// Converter para slice de tags EC2
	var tags []ec2types.Tag
	for k, v := range tagMap {
		tags = append(tags, ec2types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}

	return tags
}

func (r *SetupEKSReconciler) buildIAMTags(setup *infrav1alpha1.SetupEKS, name string) []iamtypes.Tag {
	// Usar map para evitar duplicação de tags (case-insensitive)
	tagMap := make(map[string]string)

	// Tags padrão
	tagMap["Name"] = name
	tagMap["ManagedBy"] = "infra-operator"
	tagMap["SetupEKS"] = setup.Name

	// Tags do usuário (podem sobrescrever as padrões)
	for k, v := range setup.Spec.Tags {
		tagMap[k] = v
	}

	// Converter para slice de tags IAM
	var tags []iamtypes.Tag
	for k, v := range tagMap {
		tags = append(tags, iamtypes.Tag{Key: aws.String(k), Value: aws.String(v)})
	}

	return tags
}

func (r *SetupEKSReconciler) buildStringTags(setup *infrav1alpha1.SetupEKS) map[string]string {
	tags := map[string]string{
		"ManagedBy": "infra-operator",
		"SetupEKS":  setup.Name,
	}

	for k, v := range setup.Spec.Tags {
		tags[k] = v
	}

	return tags
}

func (r *SetupEKSReconciler) calculateSubnetCIDRs(vpcCIDR string, count int, startIndex int) []string {
	// Ex: 10.0.0.0/16 -> 10.0.1.0/24, 10.0.2.0/24, ...
	parts := strings.Split(vpcCIDR, ".")
	if len(parts) < 3 {
		return nil
	}

	prefix := parts[0] + "." + parts[1]
	var cidrs []string

	for i := 0; i < count; i++ {
		thirdOctet := startIndex + i + 1 // 1, 2, 3... or 11, 12, 13...
		cidrs = append(cidrs, fmt.Sprintf("%s.%d.0/24", prefix, thirdOctet))
	}

	return cidrs
}

func (r *SetupEKSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.SetupEKS{}).
		Complete(r)
}
