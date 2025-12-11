package mapper

import (
	"time"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/ecr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CRToDomainECRRepository converts a CR to a domain ECR repository
func CRToDomainECRRepository(cr *infrav1alpha1.ECRRepository) *ecr.Repository {
	repository := &ecr.Repository{
		RepositoryName:     cr.Spec.RepositoryName,
		ImageTagMutability: cr.Spec.ImageTagMutability,
		ScanOnPush:         cr.Spec.ScanOnPush,
		Tags:               cr.Spec.Tags,
		DeletionPolicy:     cr.Spec.DeletionPolicy,
	}

	// Map encryption configuration
	if cr.Spec.EncryptionConfiguration != nil {
		repository.EncryptionType = cr.Spec.EncryptionConfiguration.EncryptionType
		repository.KmsKey = cr.Spec.EncryptionConfiguration.KmsKey
	}

	// Map lifecycle policy
	if cr.Spec.LifecyclePolicy != nil {
		repository.LifecyclePolicyText = cr.Spec.LifecyclePolicy.PolicyText
	}

	// If status has information, populate it
	if cr.Status.RepositoryArn != "" {
		repository.RepositoryArn = cr.Status.RepositoryArn
		repository.RepositoryUri = cr.Status.RepositoryUri
		repository.RegistryId = cr.Status.RegistryId
		repository.ImageCount = cr.Status.ImageCount
	}

	if !cr.Status.CreatedAt.IsZero() {
		createdAt := cr.Status.CreatedAt.Time
		repository.CreatedAt = &createdAt
	}

	if !cr.Status.LastSyncTime.IsZero() {
		syncTime := cr.Status.LastSyncTime.Time
		repository.LastSyncTime = &syncTime
	}

	return repository
}

// UpdateCRStatusFromECRRepository updates the CR status from a domain ECR repository
func UpdateCRStatusFromECRRepository(cr *infrav1alpha1.ECRRepository, repository *ecr.Repository) {
	cr.Status.RepositoryArn = repository.RepositoryArn
	cr.Status.RepositoryUri = repository.RepositoryUri
	cr.Status.RegistryId = repository.RegistryId
	cr.Status.ImageCount = repository.ImageCount

	// Set ready status
	cr.Status.Ready = repository.IsReady()

	// Update created at time
	if repository.CreatedAt != nil {
		cr.Status.CreatedAt = metav1.NewTime(*repository.CreatedAt)
	}

	// Update last sync time
	if repository.LastSyncTime != nil {
		cr.Status.LastSyncTime = metav1.NewTime(*repository.LastSyncTime)
	} else {
		cr.Status.LastSyncTime = metav1.NewTime(time.Now())
	}

	// Update conditions
	updateECRConditions(cr, repository)
}

func updateECRConditions(cr *infrav1alpha1.ECRRepository, repository *ecr.Repository) {
	now := metav1.NewTime(time.Now())

	var conditionStatus metav1.ConditionStatus
	var reason, message string

	if repository.IsReady() {
		conditionStatus = metav1.ConditionTrue
		reason = "RepositoryReady"
		message = "ECR repository is ready"
	} else {
		conditionStatus = metav1.ConditionFalse
		reason = "RepositoryNotReady"
		message = "ECR repository is not yet ready"
	}

	// Create or update the Ready condition
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             conditionStatus,
		ObservedGeneration: cr.Generation,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	// Find and update or append the condition
	found := false
	for i, cond := range cr.Status.Conditions {
		if cond.Type == "Ready" {
			// Only update LastTransitionTime if status changed
			if cond.Status != conditionStatus {
				cr.Status.Conditions[i] = condition
			} else {
				condition.LastTransitionTime = cond.LastTransitionTime
				cr.Status.Conditions[i] = condition
			}
			found = true
			break
		}
	}

	if !found {
		cr.Status.Conditions = append(cr.Status.Conditions, condition)
	}
}
