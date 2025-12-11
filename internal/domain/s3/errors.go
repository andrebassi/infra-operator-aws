package s3

import "errors"

// Domain errors
var (
	ErrBucketNameRequired      = errors.New("bucket name is required")
	ErrInvalidBucketNameLength = errors.New("bucket name must be between 3 and 63 characters")
	ErrRegionRequired          = errors.New("region is required")
	ErrBucketNotFound          = errors.New("bucket not found")
	ErrBucketAlreadyExists     = errors.New("bucket already exists")
	ErrInvalidEncryption       = errors.New("invalid encryption configuration")
	ErrInvalidLifecycleRule    = errors.New("invalid lifecycle rule")
)
