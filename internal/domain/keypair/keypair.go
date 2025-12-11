package keypair

import (
	"errors"
	"time"
)

// KeyPair representa um par de chaves EC2
type KeyPair struct {
	// KeyName é o nome do par de chaves
	KeyName string

	// KeyPairID é o ID do par de chaves na AWS
	KeyPairID string

	// KeyFingerprint é a impressão digital da chave
	KeyFingerprint string

	// KeyType é o tipo da chave (rsa, ed25519)
	KeyType string

	// PublicKeyMaterial é a chave pública (para importação)
	PublicKeyMaterial string

	// PrivateKeyMaterial é a chave privada (disponível apenas quando criada)
	PrivateKeyMaterial string

	// Tags são as etiquetas do recurso
	Tags map[string]string

	// DeletionPolicy define o comportamento ao deletar
	DeletionPolicy string

	// LastSyncTime é a última vez que foi sincronizado
	LastSyncTime *time.Time
}

// SetDefaults define valores padrão para o par de chaves
func (k *KeyPair) SetDefaults() {
	if k.DeletionPolicy == "" {
		k.DeletionPolicy = "Delete"
	}
	if k.KeyType == "" {
		k.KeyType = "rsa"
	}
}

// Validate valida a configuração do par de chaves
func (k *KeyPair) Validate() error {
	if k.KeyName == "" {
		return errors.New("keyName é obrigatório")
	}
	return nil
}

// IsReady retorna true se o par de chaves existe e está disponível
func (k *KeyPair) IsReady() bool {
	return k.KeyPairID != "" && k.KeyFingerprint != ""
}

// ShouldDelete retorna true se o par de chaves deve ser deletado
func (k *KeyPair) ShouldDelete() bool {
	return k.DeletionPolicy == "Delete" && k.KeyPairID != ""
}
