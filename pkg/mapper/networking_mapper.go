package mapper

import (
	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/internetgateway"
	"infra-operator/internal/domain/natgateway"
	"infra-operator/internal/domain/routetable"
	"infra-operator/internal/domain/securitygroup"
	"infra-operator/internal/domain/subnet"
	"infra-operator/internal/domain/vpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// VPC Mappers
func CRToDomainVPC(cr *infrav1alpha1.VPC) *vpc.VPC {
	// Ensure tags map exists and add Name tag from CR metadata if not present
	tags := cr.Spec.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	if _, exists := tags["Name"]; !exists {
		tags["Name"] = cr.Name
	}

	v := &vpc.VPC{
		CidrBlock:          cr.Spec.CidrBlock,
		EnableDnsSupport:   cr.Spec.EnableDnsSupport,
		EnableDnsHostnames: cr.Spec.EnableDnsHostnames,
		InstanceTenancy:    cr.Spec.InstanceTenancy,
		Tags:               tags,
		DeletionPolicy:     cr.Spec.DeletionPolicy,
	}
	if cr.Status.VpcID != "" {
		v.VpcID = cr.Status.VpcID
		v.State = cr.Status.State
		v.IsDefault = cr.Status.IsDefault
	}
	return v
}

func DomainToStatusVPC(v *vpc.VPC, cr *infrav1alpha1.VPC) {
	now := metav1.Now()
	cr.Status.Ready = v.IsAvailable()
	cr.Status.VpcID = v.VpcID
	cr.Status.State = v.State
	cr.Status.CidrBlock = v.CidrBlock
	cr.Status.IsDefault = v.IsDefault
	cr.Status.LastSyncTime = &now
	v.LastSyncTime = &time.Time{}
	*v.LastSyncTime = now.Time
}

// Subnet Mappers
func CRToDomainSubnet(cr *infrav1alpha1.Subnet) *subnet.Subnet {
	// Ensure tags map exists and add Name tag from CR metadata if not present
	tags := cr.Spec.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	if _, exists := tags["Name"]; !exists {
		tags["Name"] = cr.Name
	}

	s := &subnet.Subnet{
		VpcID:               cr.Spec.VpcID,
		CidrBlock:           cr.Spec.CidrBlock,
		AvailabilityZone:    cr.Spec.AvailabilityZone,
		MapPublicIpOnLaunch: cr.Spec.MapPublicIpOnLaunch,
		Tags:                tags,
		DeletionPolicy:      cr.Spec.DeletionPolicy,
	}
	if cr.Status.SubnetID != "" {
		s.SubnetID = cr.Status.SubnetID
		s.State = cr.Status.State
	}
	return s
}

func DomainToStatusSubnet(s *subnet.Subnet, cr *infrav1alpha1.Subnet) {
	now := metav1.Now()
	cr.Status.Ready = s.IsAvailable()
	cr.Status.SubnetID = s.SubnetID
	cr.Status.State = s.State
	cr.Status.VpcID = s.VpcID
	cr.Status.CidrBlock = s.CidrBlock
	cr.Status.AvailabilityZone = s.AvailabilityZone
	cr.Status.AvailableIpAddressCount = s.AvailableIpAddressCount
	cr.Status.LastSyncTime = &now
}

// Internet Gateway Mappers
func CRToDomainInternetGateway(cr *infrav1alpha1.InternetGateway) *internetgateway.Gateway {
	// Ensure tags map exists and add Name tag from CR metadata if not present
	tags := cr.Spec.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	if _, exists := tags["Name"]; !exists {
		tags["Name"] = cr.Name
	}

	g := &internetgateway.Gateway{
		VpcID:          cr.Spec.VpcID,
		Tags:           tags,
		DeletionPolicy: cr.Spec.DeletionPolicy,
	}
	if cr.Status.InternetGatewayID != "" {
		g.InternetGatewayID = cr.Status.InternetGatewayID
		g.State = cr.Status.State
	}
	return g
}

func DomainToStatusInternetGateway(g *internetgateway.Gateway, cr *infrav1alpha1.InternetGateway) {
	now := metav1.Now()
	cr.Status.Ready = g.IsAttached()
	cr.Status.InternetGatewayID = g.InternetGatewayID
	cr.Status.VpcID = g.VpcID
	cr.Status.State = g.State
	cr.Status.LastSyncTime = &now
}

// NAT Gateway Mappers
func CRToDomainNATGateway(cr *infrav1alpha1.NATGateway) *natgateway.Gateway {
	// Ensure tags map exists and add Name tag from CR metadata if not present
	tags := cr.Spec.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	if _, exists := tags["Name"]; !exists {
		tags["Name"] = cr.Name
	}

	g := &natgateway.Gateway{
		SubnetID:         cr.Spec.SubnetID,
		AllocationID:     cr.Spec.AllocationID,
		ConnectivityType: cr.Spec.ConnectivityType,
		Tags:             tags,
		DeletionPolicy:   cr.Spec.DeletionPolicy,
	}
	if cr.Status.NatGatewayID != "" {
		g.NatGatewayID = cr.Status.NatGatewayID
		g.State = cr.Status.State
		g.VpcID = cr.Status.VpcID
		g.PublicIP = cr.Status.PublicIP
		g.PrivateIP = cr.Status.PrivateIP
	}
	return g
}

func DomainToStatusNATGateway(g *natgateway.Gateway, cr *infrav1alpha1.NATGateway) {
	now := metav1.Now()
	cr.Status.Ready = g.IsAvailable()
	cr.Status.NatGatewayID = g.NatGatewayID
	cr.Status.State = g.State
	cr.Status.SubnetID = g.SubnetID
	cr.Status.VpcID = g.VpcID
	cr.Status.PublicIP = g.PublicIP
	cr.Status.PrivateIP = g.PrivateIP
	cr.Status.LastSyncTime = &now
}

// Security Group Mappers
func CRToDomainSecurityGroup(cr *infrav1alpha1.SecurityGroup) *securitygroup.SecurityGroup {
	// Ensure tags map exists and add Name tag from CR metadata if not present
	tags := cr.Spec.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	if _, exists := tags["Name"]; !exists {
		tags["Name"] = cr.Name
	}

	sg := &securitygroup.SecurityGroup{
		GroupName:      cr.Spec.GroupName,
		Description:    cr.Spec.Description,
		VpcID:          cr.Spec.VpcID,
		IngressRules:   convertCRRulesToDomain(cr.Spec.IngressRules),
		EgressRules:    convertCRRulesToDomain(cr.Spec.EgressRules),
		Tags:           tags,
		DeletionPolicy: cr.Spec.DeletionPolicy,
	}
	if cr.Status.GroupID != "" {
		sg.GroupID = cr.Status.GroupID
	}
	return sg
}

func DomainToStatusSecurityGroup(sg *securitygroup.SecurityGroup, cr *infrav1alpha1.SecurityGroup) {
	now := metav1.Now()
	cr.Status.Ready = true
	cr.Status.GroupID = sg.GroupID
	cr.Status.GroupName = sg.GroupName
	cr.Status.VpcID = sg.VpcID
	cr.Status.LastSyncTime = &now
}

func convertCRRulesToDomain(crRules []infrav1alpha1.SecurityGroupRule) []securitygroup.Rule {
	rules := make([]securitygroup.Rule, len(crRules))
	for i, crRule := range crRules {
		rules[i] = securitygroup.Rule{
			IpProtocol:            crRule.IpProtocol,
			FromPort:              crRule.FromPort,
			ToPort:                crRule.ToPort,
			CidrBlocks:            crRule.CidrBlocks,
			Ipv6CidrBlocks:        crRule.Ipv6CidrBlocks,
			SourceSecurityGroupID: crRule.SourceSecurityGroupID,
			Description:           crRule.Description,
		}
	}
	return rules
}

// RouteTable Mappers
func CRToDomainRouteTable(cr *infrav1alpha1.RouteTable) *routetable.RouteTable {
	// Ensure tags map exists and add Name tag from CR metadata if not present
	tags := cr.Spec.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	if _, exists := tags["Name"]; !exists {
		tags["Name"] = cr.Name
	}

	rt := &routetable.RouteTable{
		VpcID:              cr.Spec.VpcID,
		Routes:             convertCRRoutesToDomain(cr.Spec.Routes),
		SubnetAssociations: cr.Spec.SubnetAssociations,
		Tags:               tags,
		DeletionPolicy:     cr.Spec.DeletionPolicy,
	}
	if cr.Status.RouteTableID != "" {
		rt.RouteTableID = cr.Status.RouteTableID
	}
	return rt
}

func DomainToStatusRouteTable(rt *routetable.RouteTable, cr *infrav1alpha1.RouteTable) {
	now := metav1.Now()
	cr.Status.Ready = true
	cr.Status.RouteTableID = rt.RouteTableID
	cr.Status.VpcID = rt.VpcID
	cr.Status.AssociatedSubnets = rt.AssociatedSubnets
	cr.Status.LastSyncTime = &now
}

func convertCRRoutesToDomain(crRoutes []infrav1alpha1.Route) []routetable.Route {
	routes := make([]routetable.Route, len(crRoutes))
	for i, crRoute := range crRoutes {
		routes[i] = routetable.Route{
			DestinationCidrBlock:   crRoute.DestinationCidrBlock,
			GatewayID:              crRoute.GatewayID,
			NatGatewayID:           crRoute.NatGatewayID,
			InstanceID:             crRoute.InstanceID,
			NetworkInterfaceID:     crRoute.NetworkInterfaceID,
			VpcPeeringConnectionID: crRoute.VpcPeeringConnectionID,
		}
	}
	return routes
}
