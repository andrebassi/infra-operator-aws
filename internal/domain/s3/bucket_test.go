package s3_test

import (
	"testing"

	"infra-operator/internal/domain/s3"
)

func TestBucket_Validate(t *testing.T) {
	tests := []struct {
		name    string
		bucket  *s3.Bucket
		wantErr error
	}{
		{
			name: "valid bucket",
			bucket: &s3.Bucket{
				Name:   "my-valid-bucket",
				Region: "us-east-1",
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			bucket: &s3.Bucket{
				Name:   "",
				Region: "us-east-1",
			},
			wantErr: s3.ErrBucketNameRequired,
		},
		{
			name: "name too short",
			bucket: &s3.Bucket{
				Name:   "ab",
				Region: "us-east-1",
			},
			wantErr: s3.ErrInvalidBucketNameLength,
		},
		{
			name: "name too long",
			bucket: &s3.Bucket{
				Name:   "this-is-a-very-long-bucket-name-that-exceeds-the-maximum-allowed-length-of-63-characters",
				Region: "us-east-1",
			},
			wantErr: s3.ErrInvalidBucketNameLength,
		},
		{
			name: "empty region",
			bucket: &s3.Bucket{
				Name:   "valid-name",
				Region: "",
			},
			wantErr: s3.ErrRegionRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bucket.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBucket_IsEncrypted(t *testing.T) {
	tests := []struct {
		name   string
		bucket *s3.Bucket
		want   bool
	}{
		{
			name: "encrypted with AES256",
			bucket: &s3.Bucket{
				Encryption: &s3.EncryptionConfig{
					Algorithm: "AES256",
				},
			},
			want: true,
		},
		{
			name: "encrypted with KMS",
			bucket: &s3.Bucket{
				Encryption: &s3.EncryptionConfig{
					Algorithm: "aws:kms",
					KMSKeyID:  "arn:aws:kms:us-east-1:123456789012:key/abc",
				},
			},
			want: true,
		},
		{
			name: "no encryption",
			bucket: &s3.Bucket{
				Encryption: nil,
			},
			want: false,
		},
		{
			name: "encryption config without algorithm",
			bucket: &s3.Bucket{
				Encryption: &s3.EncryptionConfig{
					Algorithm: "",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bucket.IsEncrypted(); got != tt.want {
				t.Errorf("IsEncrypted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBucket_IsVersioned(t *testing.T) {
	tests := []struct {
		name   string
		bucket *s3.Bucket
		want   bool
	}{
		{
			name: "versioning enabled",
			bucket: &s3.Bucket{
				Versioning: &s3.VersioningConfig{
					Enabled: true,
				},
			},
			want: true,
		},
		{
			name: "versioning disabled",
			bucket: &s3.Bucket{
				Versioning: &s3.VersioningConfig{
					Enabled: false,
				},
			},
			want: false,
		},
		{
			name: "no versioning config",
			bucket: &s3.Bucket{
				Versioning: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bucket.IsVersioned(); got != tt.want {
				t.Errorf("IsVersioned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBucket_HasPublicAccessBlocked(t *testing.T) {
	tests := []struct {
		name   string
		bucket *s3.Bucket
		want   bool
	}{
		{
			name: "fully blocked",
			bucket: &s3.Bucket{
				PublicAccessBlock: &s3.PublicAccessBlockConfig{
					BlockPublicAcls:       true,
					IgnorePublicAcls:      true,
					BlockPublicPolicy:     true,
					RestrictPublicBuckets: true,
				},
			},
			want: true,
		},
		{
			name: "partially blocked",
			bucket: &s3.Bucket{
				PublicAccessBlock: &s3.PublicAccessBlockConfig{
					BlockPublicAcls:       true,
					IgnorePublicAcls:      false,
					BlockPublicPolicy:     true,
					RestrictPublicBuckets: true,
				},
			},
			want: false,
		},
		{
			name: "no public access block",
			bucket: &s3.Bucket{
				PublicAccessBlock: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bucket.HasPublicAccessBlocked(); got != tt.want {
				t.Errorf("HasPublicAccessBlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}
