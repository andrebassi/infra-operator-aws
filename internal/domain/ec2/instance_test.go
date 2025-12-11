package ec2_test
import ("testing"; "infra-operator/internal/domain/ec2")
func TestInstance_Validate(t *testing.T) {
	if err := (&ec2.Instance{InstanceName: "test", ImageID: "ami-123", InstanceType: "t3.micro"}).Validate(); err != nil {t.Error(err)}
	if err := (&ec2.Instance{}).Validate(); err == nil {t.Error("expected error")}
}
func TestInstance_SetDefaults(t *testing.T) {
	i := &ec2.Instance{}; i.SetDefaults()
	if i.DeletionPolicy != "Delete" || i.Tags == nil {t.Error("SetDefaults failed")}
}
func TestInstance_ShouldDelete(t *testing.T) {
	if !(&ec2.Instance{DeletionPolicy: "Delete"}).ShouldDelete() {t.Error("failed")}
}
