package acm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsacm "github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"

	"infra-operator/internal/domain/acm"
)

type Repository struct {
	client *awsacm.Client
}

func NewRepository(cfg aws.Config) *Repository {
	return &Repository{client: awsacm.NewFromConfig(cfg)}
}

func (r *Repository) Request(ctx context.Context, cert *acm.Certificate) error {
	input := &awsacm.RequestCertificateInput{
		DomainName:              aws.String(cert.DomainName),
		ValidationMethod:        types.ValidationMethod(cert.ValidationMethod),
		SubjectAlternativeNames: cert.SubjectAlternativeNames,
	}

	if len(cert.Tags) > 0 {
		var tags []types.Tag
		for k, v := range cert.Tags {
			tags = append(tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
		}
		input.Tags = tags
	}

	output, err := r.client.RequestCertificate(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to request certificate: %w", err)
	}

	cert.CertificateARN = aws.ToString(output.CertificateArn)
	return nil
}

func (r *Repository) Describe(ctx context.Context, certARN string) (*acm.Certificate, error) {
	input := &awsacm.DescribeCertificateInput{CertificateArn: aws.String(certARN)}
	output, err := r.client.DescribeCertificate(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe certificate: %w", err)
	}

	cert := &acm.Certificate{
		CertificateARN: certARN,
		DomainName:     aws.ToString(output.Certificate.DomainName),
		Status:         string(output.Certificate.Status),
	}

	for _, opt := range output.Certificate.DomainValidationOptions {
		if opt.ResourceRecord != nil {
			cert.ValidationRecords = append(cert.ValidationRecords, acm.ValidationRecord{
				DomainName:          aws.ToString(opt.DomainName),
				ResourceRecordName:  aws.ToString(opt.ResourceRecord.Name),
				ResourceRecordType:  string(opt.ResourceRecord.Type),
				ResourceRecordValue: aws.ToString(opt.ResourceRecord.Value),
			})
		}
	}

	return cert, nil
}

func (r *Repository) Delete(ctx context.Context, certARN string) error {
	input := &awsacm.DeleteCertificateInput{CertificateArn: aws.String(certARN)}
	_, err := r.client.DeleteCertificate(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}
	return nil
}
