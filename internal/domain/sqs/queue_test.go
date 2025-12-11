package sqs_test
import ("testing"; "infra-operator/internal/domain/sqs")
func TestQueue_Validate(t *testing.T) {
	tests := []struct {name string; q *sqs.Queue; wantErr error}{
		{"valid", &sqs.Queue{Name: "test"}, nil},
		{"no name", &sqs.Queue{}, sqs.ErrInvalidQueueName},
	}
	for _, tt := range tests {t.Run(tt.name, func(t *testing.T) {if err := tt.q.Validate(); err != tt.wantErr {t.Errorf("got %v, want %v", err, tt.wantErr)}})}
}
