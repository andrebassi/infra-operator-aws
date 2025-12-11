package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/alb"
)

// CRToDomainALB converts ALB CR to domain model
func CRToDomainALB(cr *infrav1alpha1.ALB) *alb.LoadBalancer {
	lb := &alb.LoadBalancer{
		LoadBalancerName:         cr.Spec.LoadBalancerName,
		Scheme:                   cr.Spec.Scheme,
		Subnets:                  cr.Spec.Subnets,
		SecurityGroups:           cr.Spec.SecurityGroups,
		IPAddressType:            cr.Spec.IPAddressType,
		EnableDeletionProtection: cr.Spec.EnableDeletionProtection,
		EnableHttp2:              cr.Spec.EnableHttp2,
		EnableWafFailOpen:        cr.Spec.EnableWafFailOpen,
		IdleTimeout:              cr.Spec.IdleTimeout,
		Tags:                     cr.Spec.Tags,
		DeletionPolicy:           cr.Spec.DeletionPolicy,
	}

	// Copy status fields if present
	if cr.Status.LoadBalancerARN != "" {
		lb.LoadBalancerARN = cr.Status.LoadBalancerARN
		lb.DNSName = cr.Status.DNSName
		lb.State = cr.Status.State
		lb.VpcID = cr.Status.VpcID
		lb.CanonicalHostedZoneID = cr.Status.CanonicalHostedZoneID
	}

	return lb
}

// DomainToStatusALB updates CR status from domain model
func DomainToStatusALB(lb *alb.LoadBalancer, cr *infrav1alpha1.ALB) {
	cr.Status.Ready = lb.IsActive()
	cr.Status.LoadBalancerARN = lb.LoadBalancerARN
	cr.Status.DNSName = lb.DNSName
	cr.Status.State = lb.State
	cr.Status.VpcID = lb.VpcID
	cr.Status.CanonicalHostedZoneID = lb.CanonicalHostedZoneID

	now := metav1.NewTime(time.Now())
	cr.Status.LastSyncTime = &now
}
