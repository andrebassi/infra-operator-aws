package mapper

import (
	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/secretsmanager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CRToDomainSecretsManagerSecret(cr *infrav1alpha1.SecretsManagerSecret, secretValue string) *secretsmanager.Secret {
	secret := &secretsmanager.Secret{
		SecretName:             cr.Spec.SecretName,
		Description:            cr.Spec.Description,
		SecretString:           secretValue,
		KmsKeyId:               cr.Spec.KmsKeyId,
		RotationEnabled:        cr.Spec.RotationEnabled,
		RotationLambdaARN:      cr.Spec.RotationLambdaARN,
		AutomaticallyAfterDays: cr.Spec.AutomaticallyAfterDays,
		Tags:                   cr.Spec.Tags,
		DeletionPolicy:         cr.Spec.DeletionPolicy,
		RecoveryWindowInDays:   cr.Spec.RecoveryWindowInDays,
	}

	if cr.Status.ARN != "" {
		secret.ARN = cr.Status.ARN
	}
	if cr.Status.VersionId != "" {
		secret.VersionId = cr.Status.VersionId
	}

	return secret
}

func DomainToStatusSecretsManagerSecret(secret *secretsmanager.Secret, cr *infrav1alpha1.SecretsManagerSecret) {
	cr.Status.Ready = true
	cr.Status.ARN = secret.ARN
	cr.Status.VersionId = secret.VersionId

	if secret.CreatedAt != nil {
		cr.Status.CreatedDate = &metav1.Time{Time: *secret.CreatedAt}
	}
	if secret.LastSyncTime != nil {
		cr.Status.LastChangedDate = &metav1.Time{Time: *secret.LastSyncTime}
	}
}
