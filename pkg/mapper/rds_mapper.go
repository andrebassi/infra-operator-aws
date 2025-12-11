package mapper

import (
	"time"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/rds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CRToDomainRDSInstance converts a CR to a domain RDS instance
func CRToDomainRDSInstance(cr *infrav1alpha1.RDSInstance) *rds.DBInstance {
	instance := &rds.DBInstance{
		DBInstanceIdentifier:  cr.Spec.DBInstanceIdentifier,
		Engine:                cr.Spec.Engine,
		EngineVersion:         cr.Spec.EngineVersion,
		DBInstanceClass:       cr.Spec.DBInstanceClass,
		AllocatedStorage:      cr.Spec.AllocatedStorage,
		MasterUsername:        cr.Spec.MasterUsername,
		DBName:                cr.Spec.DBName,
		Port:                  cr.Spec.Port,
		MultiAZ:               cr.Spec.MultiAZ,
		PubliclyAccessible:    cr.Spec.PubliclyAccessible,
		StorageEncrypted:      cr.Spec.StorageEncrypted,
		BackupRetentionPeriod: cr.Spec.BackupRetentionPeriod,
		PreferredBackupWindow: cr.Spec.PreferredBackupWindow,
		Tags:                  cr.Spec.Tags,
		SkipFinalSnapshot:     cr.Spec.SkipFinalSnapshot,
		DeletionPolicy:        cr.Spec.DeletionPolicy,
	}

	// Get password from direct field or secret reference
	if cr.Spec.MasterUserPassword != "" {
		instance.MasterPassword = cr.Spec.MasterUserPassword
	}
	// Note: Password from secret will be handled in the controller

	// If status has information, populate it
	if cr.Status.DBInstanceArn != "" {
		instance.DBInstanceArn = cr.Status.DBInstanceArn
		instance.Status = cr.Status.Status
		instance.Endpoint = cr.Status.Endpoint
	}

	if !cr.Status.LastSyncTime.IsZero() {
		syncTime := cr.Status.LastSyncTime.Time
		instance.LastSyncTime = &syncTime
	}

	return instance
}

// UpdateCRStatus updates the CR status from a domain RDS instance
func UpdateCRStatusFromRDSInstance(cr *infrav1alpha1.RDSInstance, instance *rds.DBInstance) {
	cr.Status.DBInstanceArn = instance.DBInstanceArn
	cr.Status.Endpoint = instance.Endpoint
	cr.Status.Port = instance.Port
	cr.Status.Status = instance.Status
	cr.Status.EngineVersion = instance.EngineVersion
	cr.Status.AllocatedStorage = instance.AllocatedStorage

	// Set ready status based on instance status
	cr.Status.Ready = instance.IsAvailable()

	// Update last sync time
	if instance.LastSyncTime != nil {
		cr.Status.LastSyncTime = metav1.NewTime(*instance.LastSyncTime)
	} else {
		cr.Status.LastSyncTime = metav1.NewTime(time.Now())
	}

	// Update conditions based on status
	updateRDSConditions(cr, instance)
}

func updateRDSConditions(cr *infrav1alpha1.RDSInstance, instance *rds.DBInstance) {
	now := metav1.NewTime(time.Now())

	// Determine condition status based on instance status
	var conditionStatus metav1.ConditionStatus
	var reason, message string

	switch instance.Status {
	case "available":
		conditionStatus = metav1.ConditionTrue
		reason = "InstanceAvailable"
		message = "RDS instance is available and ready"
	case "creating":
		conditionStatus = metav1.ConditionFalse
		reason = "InstanceCreating"
		message = "RDS instance is being created"
	case "modifying":
		conditionStatus = metav1.ConditionFalse
		reason = "InstanceModifying"
		message = "RDS instance is being modified"
	case "backing-up":
		conditionStatus = metav1.ConditionTrue
		reason = "InstanceBackingUp"
		message = "RDS instance is performing a backup"
	case "deleting":
		conditionStatus = metav1.ConditionFalse
		reason = "InstanceDeleting"
		message = "RDS instance is being deleted"
	case "failed":
		conditionStatus = metav1.ConditionFalse
		reason = "InstanceFailed"
		message = "RDS instance is in a failed state"
	default:
		conditionStatus = metav1.ConditionUnknown
		reason = "UnknownStatus"
		message = "RDS instance status is " + instance.Status
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
