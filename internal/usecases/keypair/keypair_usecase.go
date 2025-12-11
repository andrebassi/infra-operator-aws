package keypair

import (
	"context"
	"fmt"

	"infra-operator/internal/domain/keypair"
	"infra-operator/internal/ports"
)

// KeyPairUseCase implementa os casos de uso de par de chaves
type KeyPairUseCase struct {
	repo ports.KeyPairRepository
}

// NewKeyPairUseCase cria uma nova instância do caso de uso
func NewKeyPairUseCase(repo ports.KeyPairRepository) *KeyPairUseCase {
	return &KeyPairUseCase{repo: repo}
}

// SyncKeyPair sincroniza o estado desejado do par de chaves com a AWS
func (uc *KeyPairUseCase) SyncKeyPair(ctx context.Context, kp *keypair.KeyPair) error {
	kp.SetDefaults()
	if err := kp.Validate(); err != nil {
		return fmt.Errorf("falha na validação: %w", err)
	}

	// Verifica se o par de chaves já existe
	exists, err := uc.repo.Exists(ctx, kp.KeyName)
	if err != nil {
		return err
	}

	if exists {
		// Par de chaves existe, obtém o estado atual
		current, err := uc.repo.Get(ctx, kp.KeyName)
		if err != nil {
			return err
		}
		kp.KeyPairID = current.KeyPairID
		kp.KeyFingerprint = current.KeyFingerprint
		kp.KeyType = current.KeyType

		// Aplica tags se necessário
		if len(kp.Tags) > 0 && kp.KeyPairID != "" {
			uc.repo.TagResource(ctx, kp.KeyPairID, kp.Tags)
		}
		return nil
	}

	// Cria ou importa o par de chaves
	if kp.PublicKeyMaterial != "" {
		// Importa chave pública existente
		return uc.repo.Import(ctx, kp)
	}

	// Cria novo par de chaves (a AWS gera a chave)
	return uc.repo.Create(ctx, kp)
}

// DeleteKeyPair deleta o par de chaves da AWS
func (uc *KeyPairUseCase) DeleteKeyPair(ctx context.Context, kp *keypair.KeyPair) error {
	if !kp.ShouldDelete() {
		return nil
	}
	return uc.repo.Delete(ctx, kp.KeyName)
}
