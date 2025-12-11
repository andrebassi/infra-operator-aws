package mapper

import (
	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/kms"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CRToDomainKMSKey(cr *infrav1alpha1.KMSKey) *kms.Key {
	key := &kms.Key{
		Description:         cr.Spec.Description,
		KeyUsage:            cr.Spec.KeyUsage,
		KeySpec:             cr.Spec.KeySpec,
		MultiRegion:         cr.Spec.MultiRegion,
		EnableKeyRotation:   cr.Spec.EnableKeyRotation,
		Enabled:             cr.Spec.Enabled,
		KeyPolicy:           cr.Spec.KeyPolicy,
		Tags:                cr.Spec.Tags,
		DeletionPolicy:      cr.Spec.DeletionPolicy,
		PendingWindowInDays: cr.Spec.PendingWindowInDays,
	}

	// If status has KeyId, use it
	if cr.Status.KeyId != "" {
		key.KeyId = cr.Status.KeyId
	}
	if cr.Status.Arn != "" {
		key.Arn = cr.Status.Arn
	}

	return key
}

func DomainToStatusKMSKey(key *kms.Key, cr *infrav1alpha1.KMSKey) {
	cr.Status.Ready = true
	cr.Status.KeyId = key.KeyId
	cr.Status.Arn = key.Arn
	cr.Status.KeyState = key.KeyState

	if key.CreatedAt != nil {
		cr.Status.CreationDate = &metav1.Time{Time: *key.CreatedAt}
	}
	if key.LastSyncTime != nil {
		cr.Status.LastSyncTime = &metav1.Time{Time: *key.LastSyncTime}
	}
}
