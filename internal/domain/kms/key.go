package kms

import (
	"errors"
	"time"
)

var (
	ErrInvalidPendingWindow = errors.New("pending window must be between 7 and 30 days")
)

type Key struct {
	KeyId               string
	Arn                 string
	Description         string
	KeyUsage            string
	KeySpec             string
	MultiRegion         bool
	EnableKeyRotation   bool
	Enabled             bool
	KeyPolicy           string
	Tags                map[string]string
	DeletionPolicy      string
	PendingWindowInDays int32
	KeyState            string
	CreatedAt           *time.Time
	LastSyncTime        *time.Time
}

func (k *Key) SetDefaults() {
	if k.KeyUsage == "" {
		k.KeyUsage = "ENCRYPT_DECRYPT"
	}
	if k.KeySpec == "" {
		k.KeySpec = "SYMMETRIC_DEFAULT"
	}
	if k.DeletionPolicy == "" {
		k.DeletionPolicy = "Retain"
	}
	if k.PendingWindowInDays == 0 {
		k.PendingWindowInDays = 30
	}
	if k.Tags == nil {
		k.Tags = make(map[string]string)
	}
}

func (k *Key) Validate() error {
	if k.PendingWindowInDays < 7 || k.PendingWindowInDays > 30 {
		return ErrInvalidPendingWindow
	}
	return nil
}

func (k *Key) ShouldDelete() bool {
	return k.DeletionPolicy == "Delete"
}

func (k *Key) IsSymmetric() bool {
	return k.KeySpec == "SYMMETRIC_DEFAULT"
}
