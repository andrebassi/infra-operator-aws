package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/cloudfront"
)

func CRToDomainCloudFront(cr *infrav1alpha1.CloudFront) *cloudfront.Distribution {
	dist := &cloudfront.Distribution{
		Comment:           cr.Spec.Comment,
		DefaultRootObject: cr.Spec.DefaultRootObject,
		Enabled:           cr.Spec.Enabled,
		PriceClass:        cr.Spec.PriceClass,
		Aliases:           cr.Spec.Aliases,
		Tags:              cr.Spec.Tags,
		DeletionPolicy:    cr.Spec.DeletionPolicy,
	}

	// Map origins
	for _, o := range cr.Spec.Origins {
		origin := cloudfront.Origin{
			ID:            o.ID,
			DomainName:    o.DomainName,
			OriginPath:    o.OriginPath,
			CustomHeaders: o.CustomHeaders,
		}
		if o.S3OriginConfig != nil {
			origin.S3OriginConfig = &cloudfront.S3OriginConfig{
				OriginAccessIdentity: o.S3OriginConfig.OriginAccessIdentity,
			}
		}
		if o.CustomOriginConfig != nil {
			origin.CustomOriginConfig = &cloudfront.CustomOriginConfig{
				HTTPPort:             o.CustomOriginConfig.HTTPPort,
				HTTPSPort:            o.CustomOriginConfig.HTTPSPort,
				OriginProtocolPolicy: o.CustomOriginConfig.OriginProtocolPolicy,
			}
		}
		dist.Origins = append(dist.Origins, origin)
	}

	// Map default cache behavior
	dist.DefaultCacheBehavior = cloudfront.CacheBehavior{
		TargetOriginID:       cr.Spec.DefaultCacheBehavior.TargetOriginID,
		ViewerProtocolPolicy: cr.Spec.DefaultCacheBehavior.ViewerProtocolPolicy,
		AllowedMethods:       cr.Spec.DefaultCacheBehavior.AllowedMethods,
		CachedMethods:        cr.Spec.DefaultCacheBehavior.CachedMethods,
		Compress:             cr.Spec.DefaultCacheBehavior.Compress,
		MinTTL:               cr.Spec.DefaultCacheBehavior.MinTTL,
		MaxTTL:               cr.Spec.DefaultCacheBehavior.MaxTTL,
		DefaultTTL:           cr.Spec.DefaultCacheBehavior.DefaultTTL,
	}

	// Map viewer certificate
	if cr.Spec.ViewerCertificate != nil {
		dist.ViewerCertificate = &cloudfront.ViewerCertificate{
			ACMCertificateARN:            cr.Spec.ViewerCertificate.ACMCertificateARN,
			CloudFrontDefaultCertificate: cr.Spec.ViewerCertificate.CloudFrontDefaultCertificate,
			MinimumProtocolVersion:       cr.Spec.ViewerCertificate.MinimumProtocolVersion,
			SSLSupportMethod:             cr.Spec.ViewerCertificate.SSLSupportMethod,
		}
	}

	if cr.Status.DistributionID != "" {
		dist.DistributionID = cr.Status.DistributionID
		dist.DomainName = cr.Status.DomainName
		dist.Status = cr.Status.Status
	}

	return dist
}

func DomainToStatusCloudFront(dist *cloudfront.Distribution, cr *infrav1alpha1.CloudFront) {
	cr.Status.Ready = dist.IsReady()
	cr.Status.DistributionID = dist.DistributionID
	cr.Status.DomainName = dist.DomainName
	cr.Status.Status = dist.Status

	now := metav1.NewTime(time.Now())
	cr.Status.LastSyncTime = &now
}
