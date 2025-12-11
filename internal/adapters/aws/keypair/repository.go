package keypair

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"infra-operator/internal/domain/keypair"
)

// Repository implementa o repositório de par de chaves usando AWS SDK
type Repository struct {
	client *awsec2.Client
}

// NewRepository cria uma nova instância do repositório
func NewRepository(cfg aws.Config) *Repository {
	return &Repository{client: awsec2.NewFromConfig(cfg)}
}

// Exists verifica se um par de chaves existe pelo nome
func (r *Repository) Exists(ctx context.Context, keyName string) (bool, error) {
	output, err := r.client.DescribeKeyPairs(ctx, &awsec2.DescribeKeyPairsInput{
		KeyNames: []string{keyName},
	})
	if err != nil {
		// Verifica se é um erro de NotFound
		return false, nil
	}
	return len(output.KeyPairs) > 0, nil
}

// Get recupera um par de chaves pelo nome
func (r *Repository) Get(ctx context.Context, keyName string) (*keypair.KeyPair, error) {
	output, err := r.client.DescribeKeyPairs(ctx, &awsec2.DescribeKeyPairsInput{
		KeyNames: []string{keyName},
	})
	if err != nil {
		return nil, err
	}
	if len(output.KeyPairs) == 0 {
		return nil, fmt.Errorf("par de chaves não encontrado: %s", keyName)
	}

	kp := output.KeyPairs[0]
	return &keypair.KeyPair{
		KeyName:        aws.ToString(kp.KeyName),
		KeyPairID:      aws.ToString(kp.KeyPairId),
		KeyFingerprint: aws.ToString(kp.KeyFingerprint),
		KeyType:        string(kp.KeyType),
	}, nil
}

// Create cria um novo par de chaves (a AWS gera a chave)
func (r *Repository) Create(ctx context.Context, kp *keypair.KeyPair) error {
	output, err := r.client.CreateKeyPair(ctx, &awsec2.CreateKeyPairInput{
		KeyName: aws.String(kp.KeyName),
		KeyType: types.KeyType(kp.KeyType),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeKeyPair,
				Tags:         convertToEC2Tags(kp.Tags),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("falha ao criar par de chaves: %w", err)
	}

	kp.KeyPairID = aws.ToString(output.KeyPairId)
	kp.KeyFingerprint = aws.ToString(output.KeyFingerprint)
	kp.PrivateKeyMaterial = aws.ToString(output.KeyMaterial)

	return nil
}

// Import importa um par de chaves com chave pública existente
func (r *Repository) Import(ctx context.Context, kp *keypair.KeyPair) error {
	output, err := r.client.ImportKeyPair(ctx, &awsec2.ImportKeyPairInput{
		KeyName:           aws.String(kp.KeyName),
		PublicKeyMaterial: []byte(kp.PublicKeyMaterial),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeKeyPair,
				Tags:         convertToEC2Tags(kp.Tags),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("falha ao importar par de chaves: %w", err)
	}

	kp.KeyPairID = aws.ToString(output.KeyPairId)
	kp.KeyFingerprint = aws.ToString(output.KeyFingerprint)

	return nil
}

// Delete deleta um par de chaves pelo nome
func (r *Repository) Delete(ctx context.Context, keyName string) error {
	_, err := r.client.DeleteKeyPair(ctx, &awsec2.DeleteKeyPairInput{
		KeyName: aws.String(keyName),
	})
	if err != nil {
		return fmt.Errorf("falha ao deletar par de chaves: %w", err)
	}
	return nil
}

// TagResource adiciona tags a um par de chaves
func (r *Repository) TagResource(ctx context.Context, keyPairID string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	_, err := r.client.CreateTags(ctx, &awsec2.CreateTagsInput{
		Resources: []string{keyPairID},
		Tags:      convertToEC2Tags(tags),
	})
	return err
}

// convertToEC2Tags converte um mapa de tags para o formato AWS EC2
func convertToEC2Tags(tags map[string]string) []types.Tag {
	ec2Tags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		ec2Tags = append(ec2Tags, types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return ec2Tags
}
