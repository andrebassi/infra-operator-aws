package cloudfront

import "errors"

var (
	ErrInvalidOrigins      = errors.New("invalid origins: must have at least one origin")
	ErrInvalidPriceClass   = errors.New("invalid price class")
	ErrInvalidDeletionPolicy = errors.New("invalid deletion policy: must be Delete, Retain, or Orphan")
)

const (
	PriceClassAll = "PriceClass_All"
	PriceClass100 = "PriceClass_100"
	PriceClass200 = "PriceClass_200"

	DeletionPolicyDelete = "Delete"
	DeletionPolicyRetain = "Retain"
	DeletionPolicyOrphan = "Orphan"
)

// Distribution represents a CloudFront distribution
type Distribution struct {
	Comment              string
	DefaultRootObject    string
	Origins              []Origin
	DefaultCacheBehavior CacheBehavior
	CacheBehaviors       []CacheBehavior
	Enabled              bool
	PriceClass           string
	ViewerCertificate    *ViewerCertificate
	Aliases              []string
	Tags                 map[string]string
	DeletionPolicy       string

	// Output fields from AWS
	DistributionID string
	DomainName     string
	Status         string
	ETag           string
}

type Origin struct {
	ID                 string
	DomainName         string
	OriginPath         string
	CustomHeaders      map[string]string
	S3OriginConfig     *S3OriginConfig
	CustomOriginConfig *CustomOriginConfig
}

type S3OriginConfig struct {
	OriginAccessIdentity string
}

type CustomOriginConfig struct {
	HTTPPort             int32
	HTTPSPort            int32
	OriginProtocolPolicy string
}

type CacheBehavior struct {
	PathPattern          string
	TargetOriginID       string
	ViewerProtocolPolicy string
	AllowedMethods       []string
	CachedMethods        []string
	Compress             bool
	MinTTL               int64
	MaxTTL               int64
	DefaultTTL           int64
}

type ViewerCertificate struct {
	ACMCertificateARN            string
	CloudFrontDefaultCertificate bool
	MinimumProtocolVersion       string
	SSLSupportMethod             string
}

func (d *Distribution) Validate() error {
	if len(d.Origins) == 0 {
		return ErrInvalidOrigins
	}

	if d.PriceClass != "" {
		if d.PriceClass != PriceClassAll && d.PriceClass != PriceClass100 && d.PriceClass != PriceClass200 {
			return ErrInvalidPriceClass
		}
	}

	if d.DeletionPolicy != "" {
		if d.DeletionPolicy != DeletionPolicyDelete && d.DeletionPolicy != DeletionPolicyRetain && d.DeletionPolicy != DeletionPolicyOrphan {
			return ErrInvalidDeletionPolicy
		}
	}

	return nil
}

func (d *Distribution) SetDefaults() {
	if d.PriceClass == "" {
		d.PriceClass = PriceClass100
	}

	if d.DeletionPolicy == "" {
		d.DeletionPolicy = DeletionPolicyDelete
	}
}

func (d *Distribution) IsReady() bool {
	return d.DistributionID != "" && d.Status == "Deployed"
}
