package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/elasticip"
)

// CRToDomainElasticIP converts ElasticIP CR to domain model
func CRToDomainElasticIP(cr *infrav1alpha1.ElasticIP) *elasticip.Address {
	addr := &elasticip.Address{
		Domain:                cr.Spec.Domain,
		NetworkBorderGroup:    cr.Spec.NetworkBorderGroup,
		PublicIpv4Pool:        cr.Spec.PublicIpv4Pool,
		CustomerOwnedIpv4Pool: cr.Spec.CustomerOwnedIpv4Pool,
		Tags:                  cr.Spec.Tags,
		DeletionPolicy:        cr.Spec.DeletionPolicy,
	}

	// Copy status fields if present
	if cr.Status.AllocationID != "" {
		addr.AllocationID = cr.Status.AllocationID
		addr.PublicIP = cr.Status.PublicIP
		addr.AssociationID = cr.Status.AssociationID
		addr.InstanceID = cr.Status.InstanceID
		addr.NetworkInterfaceID = cr.Status.NetworkInterfaceID
		addr.PrivateIPAddress = cr.Status.PrivateIPAddress
	}

	return addr
}

// DomainToStatusElasticIP updates CR status from domain model
func DomainToStatusElasticIP(addr *elasticip.Address, cr *infrav1alpha1.ElasticIP) {
	cr.Status.Ready = addr.IsAllocated()
	cr.Status.AllocationID = addr.AllocationID
	cr.Status.PublicIP = addr.PublicIP
	cr.Status.AssociationID = addr.AssociationID
	cr.Status.InstanceID = addr.InstanceID
	cr.Status.NetworkInterfaceID = addr.NetworkInterfaceID
	cr.Status.PrivateIPAddress = addr.PrivateIPAddress
	cr.Status.Domain = addr.Domain

	now := metav1.NewTime(time.Now())
	cr.Status.LastSyncTime = &now
}
