package ports

import (
	"context"

	"infra-operator/internal/domain/keypair"
)

// KeyPairRepository define a interface para operações de par de chaves EC2
type KeyPairRepository interface {
	// Exists verifica se um par de chaves existe pelo nome
	Exists(ctx context.Context, keyName string) (bool, error)

	// Get recupera um par de chaves pelo nome
	Get(ctx context.Context, keyName string) (*keypair.KeyPair, error)

	// Create cria um novo par de chaves (a AWS gera a chave)
	Create(ctx context.Context, kp *keypair.KeyPair) error

	// Import importa um par de chaves com material de chave pública existente
	Import(ctx context.Context, kp *keypair.KeyPair) error

	// Delete deleta um par de chaves pelo nome
	Delete(ctx context.Context, keyName string) error

	// TagResource adiciona tags a um par de chaves
	TagResource(ctx context.Context, keyPairID string, tags map[string]string) error
}
