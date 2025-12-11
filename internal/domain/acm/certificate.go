package acm

import (
	"errors"
	"time"
)

var (
	ErrInvalidDomainName      = errors.New("domain name cannot be empty")
	ErrInvalidValidationMethod = errors.New("validation method must be 'DNS' or 'EMAIL'")
)

type Certificate struct {
	DomainName              string
	SubjectAlternativeNames []string
	CertificateARN          string
	Status                  string
	ValidationMethod        string
	ValidationRecords       []ValidationRecord
	Tags                    map[string]string
	DeletionPolicy          string
	LastSyncTime            *time.Time
}

type ValidationRecord struct {
	DomainName          string
	ResourceRecordName  string
	ResourceRecordType  string
	ResourceRecordValue string
}

func (c *Certificate) Validate() error {
	if c.DomainName == "" {
		return ErrInvalidDomainName
	}
	if c.ValidationMethod != "" && c.ValidationMethod != "DNS" && c.ValidationMethod != "EMAIL" {
		return ErrInvalidValidationMethod
	}
	return nil
}

func (c *Certificate) SetDefaults() {
	if c.ValidationMethod == "" {
		c.ValidationMethod = "DNS"
	}
	if c.DeletionPolicy == "" {
		c.DeletionPolicy = "Delete"
	}
	if c.Tags == nil {
		c.Tags = make(map[string]string)
	}
}

func (c *Certificate) ShouldDelete() bool {
	return c.DeletionPolicy == "Delete"
}

func (c *Certificate) IsIssued() bool {
	return c.Status == "ISSUED"
}

func (c *Certificate) IsPendingValidation() bool {
	return c.Status == "PENDING_VALIDATION"
}

func (c *Certificate) IsFailed() bool {
	return c.Status == "FAILED"
}
