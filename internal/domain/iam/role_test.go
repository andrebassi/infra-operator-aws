package iam_test
import ("testing"; "infra-operator/internal/domain/iam")
func TestRole_Validate(t *testing.T) {
	if err := (&iam.Role{RoleName: "test", AssumeRolePolicyDocument: "{}"}).Validate(); err != nil {t.Error(err)}
	if err := (&iam.Role{}).Validate(); err == nil {t.Error("expected error")}
}
func TestRole_SetDefaults(t *testing.T) {
	r := &iam.Role{}; r.SetDefaults()
	if r.DeletionPolicy != "Delete" || r.Tags == nil {t.Error("failed")}
}
func TestRole_ShouldDelete(t *testing.T) {
	if !(&iam.Role{DeletionPolicy: "Delete"}).ShouldDelete() {t.Error("failed")}
}
