package kms_test
import ("testing"; "infra-operator/internal/domain/kms")
func TestKey_Validate(t *testing.T) {
	k := &kms.Key{PendingWindowInDays: 30}
	if err := k.Validate(); err != nil {t.Error(err)}
}
func TestKey_SetDefaults(t *testing.T) {
	k := &kms.Key{}; k.SetDefaults()
	if k.DeletionPolicy != "Retain" || k.Tags == nil || k.PendingWindowInDays != 30 {t.Error("failed")}
}
func TestKey_ShouldDelete(t *testing.T) {
	if !(&kms.Key{DeletionPolicy: "Delete"}).ShouldDelete() {t.Error("failed")}
}
