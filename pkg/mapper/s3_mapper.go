package mapper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/s3"
)

// CRToDomainBucket converts Kubernetes CR to domain model
func CRToDomainBucket(cr *infrav1alpha1.S3Bucket) *s3.Bucket {
	bucket := &s3.Bucket{
		Name:   cr.Spec.BucketName,
		Region: "", // Will be set from provider
		Tags:   cr.Spec.Tags,
	}

	// Versioning
	if cr.Spec.Versioning != nil {
		bucket.Versioning = &s3.VersioningConfig{
			Enabled: cr.Spec.Versioning.Enabled,
		}
	}

	// Encryption
	if cr.Spec.Encryption != nil {
		bucket.Encryption = &s3.EncryptionConfig{
			Algorithm: cr.Spec.Encryption.Algorithm,
			KMSKeyID:  cr.Spec.Encryption.KMSKeyID,
		}
	}

	// Lifecycle rules
	if len(cr.Spec.LifecycleRules) > 0 {
		bucket.LifecycleRules = make([]s3.LifecycleRule, 0, len(cr.Spec.LifecycleRules))
		for _, crRule := range cr.Spec.LifecycleRules {
			rule := s3.LifecycleRule{
				ID:      crRule.ID,
				Enabled: crRule.Enabled,
				Prefix:  crRule.Prefix,
			}

			if crRule.Expiration != nil {
				rule.Expiration = &s3.Expiration{
					Days: crRule.Expiration.Days,
				}
			}

			if len(crRule.Transitions) > 0 {
				rule.Transitions = make([]s3.Transition, 0, len(crRule.Transitions))
				for _, t := range crRule.Transitions {
					rule.Transitions = append(rule.Transitions, s3.Transition{
						Days:         t.Days,
						StorageClass: t.StorageClass,
					})
				}
			}

			bucket.LifecycleRules = append(bucket.LifecycleRules, rule)
		}
	}

	// CORS rules
	if len(cr.Spec.CORSRules) > 0 {
		bucket.CORSRules = make([]s3.CORSRule, 0, len(cr.Spec.CORSRules))
		for _, crCORS := range cr.Spec.CORSRules {
			bucket.CORSRules = append(bucket.CORSRules, s3.CORSRule{
				AllowedOrigins: crCORS.AllowedOrigins,
				AllowedMethods: crCORS.AllowedMethods,
				AllowedHeaders: crCORS.AllowedHeaders,
				ExposeHeaders:  crCORS.ExposeHeaders,
				MaxAgeSeconds:  crCORS.MaxAgeSeconds,
			})
		}
	}

	// Public access block
	if cr.Spec.PublicAccessBlock != nil {
		bucket.PublicAccessBlock = &s3.PublicAccessBlockConfig{
			BlockPublicAcls:       cr.Spec.PublicAccessBlock.BlockPublicAcls,
			IgnorePublicAcls:      cr.Spec.PublicAccessBlock.IgnorePublicAcls,
			BlockPublicPolicy:     cr.Spec.PublicAccessBlock.BlockPublicPolicy,
			RestrictPublicBuckets: cr.Spec.PublicAccessBlock.RestrictPublicBuckets,
		}
	}

	// Deletion policy
	if cr.Spec.DeletionPolicy != "" {
		bucket.DeletionPolicy = s3.DeletionPolicy(cr.Spec.DeletionPolicy)
	} else {
		bucket.DeletionPolicy = s3.DeletionPolicyDelete // default
	}

	return bucket
}

// DomainBucketToCRStatus updates CR status from domain model
func DomainBucketToCRStatus(bucket *s3.Bucket, cr *infrav1alpha1.S3Bucket) {
	cr.Status.ARN = bucket.ARN
	cr.Status.Region = bucket.Region
	cr.Status.BucketDomainName = bucket.DomainName

	if bucket.LastSyncTime != nil {
		metaTime := metav1.Time{Time: *bucket.LastSyncTime}
		cr.Status.LastSyncTime = &metaTime
	}
}

// SetBucketRegionFromProvider sets region from AWSProvider
func SetBucketRegionFromProvider(bucket *s3.Bucket, provider *infrav1alpha1.AWSProvider) {
	bucket.Region = provider.Spec.Region

	// Merge default tags from provider
	if len(provider.Spec.DefaultTags) > 0 {
		if bucket.Tags == nil {
			bucket.Tags = make(map[string]string)
		}

		for k, v := range provider.Spec.DefaultTags {
			// Don't override existing tags
			if _, exists := bucket.Tags[k]; !exists {
				bucket.Tags[k] = v
			}
		}
	}
}
