package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/nlb"
)

// CRToDomainNLB converts NLB CR to domain model
func CRToDomainNLB(cr *infrav1alpha1.NLB) *nlb.LoadBalancer {
	lb := &nlb.LoadBalancer{
		LoadBalancerName:             cr.Spec.LoadBalancerName,
		Scheme:                       cr.Spec.Scheme,
		Subnets:                      cr.Spec.Subnets,
		IPAddressType:                cr.Spec.IPAddressType,
		EnableDeletionProtection:     cr.Spec.EnableDeletionProtection,
		EnableCrossZoneLoadBalancing: cr.Spec.EnableCrossZoneLoadBalancing,
		Tags:                         cr.Spec.Tags,
		DeletionPolicy:               cr.Spec.DeletionPolicy,
	}

	if cr.Status.LoadBalancerARN != "" {
		lb.LoadBalancerARN = cr.Status.LoadBalancerARN
		lb.DNSName = cr.Status.DNSName
		lb.State = cr.Status.State
		lb.VpcID = cr.Status.VpcID
		lb.CanonicalHostedZoneID = cr.Status.CanonicalHostedZoneID
	}

	return lb
}

// DomainToStatusNLB updates CR status from domain model
func DomainToStatusNLB(lb *nlb.LoadBalancer, cr *infrav1alpha1.NLB) {
	cr.Status.Ready = lb.IsActive()
	cr.Status.LoadBalancerARN = lb.LoadBalancerARN
	cr.Status.DNSName = lb.DNSName
	cr.Status.State = lb.State
	cr.Status.VpcID = lb.VpcID
	cr.Status.CanonicalHostedZoneID = lb.CanonicalHostedZoneID

	now := metav1.NewTime(time.Now())
	cr.Status.LastSyncTime = &now
}
