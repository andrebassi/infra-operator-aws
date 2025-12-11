package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/acm"
)

func CRToDomainACM(cr *infrav1alpha1.Certificate) *acm.Certificate {
	cert := &acm.Certificate{
		DomainName:              cr.Spec.DomainName,
		SubjectAlternativeNames: cr.Spec.SubjectAlternativeNames,
		ValidationMethod:        cr.Spec.ValidationMethod,
		Tags:                    cr.Spec.Tags,
		DeletionPolicy:          cr.Spec.DeletionPolicy,
	}

	if cr.Status.CertificateARN != "" {
		cert.CertificateARN = cr.Status.CertificateARN
		cert.Status = cr.Status.Status
	}

	return cert
}

func DomainToStatusACM(cert *acm.Certificate, cr *infrav1alpha1.Certificate) {
	cr.Status.Ready = cert.IsIssued()
	cr.Status.CertificateARN = cert.CertificateARN
	cr.Status.Status = cert.Status

	var records []infrav1alpha1.CertificateValidationRecord
	for _, r := range cert.ValidationRecords {
		records = append(records, infrav1alpha1.CertificateValidationRecord{
			DomainName:          r.DomainName,
			ResourceRecordName:  r.ResourceRecordName,
			ResourceRecordType:  r.ResourceRecordType,
			ResourceRecordValue: r.ResourceRecordValue,
		})
	}
	cr.Status.ValidationRecords = records

	now := metav1.NewTime(time.Now())
	cr.Status.LastSyncTime = &now
}
