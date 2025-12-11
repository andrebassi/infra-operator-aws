package mapper

import (
	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/iam"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CRToDomainIAMRole converts an IAMRole CR to a domain Role
func CRToDomainIAMRole(cr *infrav1alpha1.IAMRole) *iam.Role {
	role := &iam.Role{
		RoleName:                 cr.Spec.RoleName,
		Description:              cr.Spec.Description,
		AssumeRolePolicyDocument: cr.Spec.AssumeRolePolicyDocument,
		MaxSessionDuration:       cr.Spec.MaxSessionDuration,
		Path:                     cr.Spec.Path,
		PermissionsBoundary:      cr.Spec.PermissionsBoundary,
		ManagedPolicyArns:        cr.Spec.ManagedPolicyArns,
		Tags:                     cr.Spec.Tags,
		DeletionPolicy:           cr.Spec.DeletionPolicy,
	}

	// Map inline policy if present
	if cr.Spec.InlinePolicy != nil {
		role.InlinePolicyName = cr.Spec.InlinePolicy.PolicyName
		role.InlinePolicyDocument = cr.Spec.InlinePolicy.PolicyDocument
	}

	// Map status fields if present
	if cr.Status.RoleArn != "" {
		role.RoleArn = cr.Status.RoleArn
	}
	if cr.Status.RoleId != "" {
		role.RoleId = cr.Status.RoleId
	}
	if cr.Status.CreatedAt != nil {
		t := cr.Status.CreatedAt.Time
		role.CreatedAt = &t
	}
	if cr.Status.LastSyncTime != nil {
		t := cr.Status.LastSyncTime.Time
		role.LastSyncTime = &t
	}

	return role
}

// DomainToStatusIAMRole updates the IAMRole CR status from a domain Role
func DomainToStatusIAMRole(role *iam.Role, cr *infrav1alpha1.IAMRole) {
	cr.Status.Ready = true
	cr.Status.RoleArn = role.RoleArn
	cr.Status.RoleId = role.RoleId

	if role.CreatedAt != nil {
		cr.Status.CreatedAt = &metav1.Time{Time: *role.CreatedAt}
	}

	if role.LastSyncTime != nil {
		cr.Status.LastSyncTime = &metav1.Time{Time: *role.LastSyncTime}
	}

	cr.Status.Message = "IAM role synchronized successfully"
}
