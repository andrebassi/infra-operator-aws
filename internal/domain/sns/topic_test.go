package sns_test
import ("testing"; "infra-operator/internal/domain/sns")
func TestTopic_Validate(t *testing.T) {
	tests := []struct {name string; topic *sns.Topic; wantErr error}{
		{"valid", &sns.Topic{Name: "test"}, nil},
		{"no name", &sns.Topic{}, sns.ErrInvalidTopicName},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.topic.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
