# Infra Operator - Arquitetura e Design

## Introdução

Este documento explica a arquitetura do **infra-operator**, um Operator Kubernetes que provisiona recursos AWS usando o padrão de Custom Resources (CRs). O operator foi desenvolvido seguindo as melhores práticas da comunidade Kubernetes e utiliza AWS SDK Go v2 para interação direta com os serviços AWS.

## Visão Geral da Arquitetura

![Operator Architecture](/img/diagrams/operator-architecture.svg)

## Componentes Principais

### 1. Custom Resource Definitions (CRDs)

CRDs definem a estrutura dos recursos customizados que o operator gerencia.

#### AWSProvider CRD

**Propósito**: Centralizar credenciais e configurações AWS.

**Campos principais**:
- `region`: Região AWS (obrigatório)
- `roleARN`: ARN do papel IAM para IRSA
- `accessKeyIDRef/secretAccessKeyRef`: Referências a Secrets para credenciais estáticas
- `defaultTags`: Tags aplicadas a todos os recursos

**Status**:
- `ready`: Indica se as credenciais são válidas
- `accountID`: ID da conta AWS
- `callerIdentity`: ARN da identidade autenticada
- `conditions`: Condições de status

**Arquivo**: `api/v1alpha1/awsprovider_types.go`

#### S3Bucket CRD

**Propósito**: Gerenciar buckets S3 com configuração completa.

**Campos principais**:
- `providerRef`: Referência ao AWSProvider
- `bucketName`: Nome do bucket (globalmente único)
- `versioning`: Configuração de versionamento
- `encryption`: Configuração de criptografia
- `lifecycleRules`: Regras de ciclo de vida
- `corsRules`: Configuração CORS
- `publicAccessBlock`: Bloqueio de acesso público
- `deletionPolicy`: Política de deleção (Delete, Retain, Orphan)

**Status**:
- `ready`: Bucket criado e configurado
- `arn`: ARN do bucket
- `region`: Região onde o bucket existe
- `bucketDomainName`: Nome de domínio do bucket

**Arquivo**: `api/v1alpha1/s3bucket_types.go`

### 2. Controllers (Reconcilers)

Controllers implementam a lógica de reconciliação - eles observam o estado desejado (spec) e o estado atual, e trabalham para convergir o estado atual ao desejado.

#### AWSProvider Controller

**Responsabilidades**:
1. Validar credenciais AWS usando STS GetCallerIdentity
2. Construir configuração AWS (aws.Config) baseada na spec
3. Atualizar status com informações da conta
4. Revalidar credenciais periodicamente (a cada 5 minutos)

**Fluxo de reconciliação**:

![AWSProvider Reconciliation Flow](/img/diagrams/awsprovider-reconciliation-flow.svg)

**Arquivo**: `controllers/awsprovider_controller.go`

#### S3Bucket Controller

**Responsabilidades**:
1. Criar bucket S3 se não existir
2. Configurar versionamento, criptografia, lifecycle, CORS, tags
3. Configurar bloqueio de acesso público
4. Gerenciar ciclo de vida (criação, atualização, deleção)
5. Implementar finalizers para cleanup controlado

**Fluxo de reconciliação**:

![S3Bucket Reconciliation Flow](/img/diagrams/s3bucket-reconciliation-flow.svg)

**Arquivo**: `controllers/s3bucket_controller.go`

### 3. AWS Helper Package

Funções auxiliares para interação com AWS.

**Funções principais**:
- `GetAWSConfigFromProvider()`: Obtém aws.Config a partir do AWSProvider CR
- `buildAWSConfig()`: Constrói configuração AWS com credenciais
- `getSecretValue()`: Lê valores de Kubernetes Secrets

**Arquivo**: `pkg/aws/provider.go`

## Padrões de Design Implementados

### 1. Reconciliation Loop

O padrão fundamental dos Operators Kubernetes. O controller observa o estado desejado (CR spec) e trabalha continuamente para fazer o estado atual convergir ao desejado.

**Características**:
- **Idempotente**: Múltiplas execuções produzem o mesmo resultado
- **Edge-triggered e Level-triggered**: Reage a mudanças e também verifica estado periodicamente
- **Tratamento de erros**: Retorna erro para requeue automático

### 2. Finalizers

Mecanismo para executar cleanup antes de deletar um CR.

**Implementação no S3Bucket**:
```go
const s3BucketFinalizer = "aws-infra-operator.runner.codes/s3bucket-finalizer"

// Ao criar/atualizar:
if !controllerutil.ContainsFinalizer(bucket, s3BucketFinalizer) {
controllerutil.AddFinalizer(bucket, s3BucketFinalizer)
r.Update(ctx, bucket)
}

// Ao deletar:
if !bucket.ObjectMeta.DeletionTimestamp.IsZero() {
if controllerutil.ContainsFinalizer(bucket, s3BucketFinalizer) {
        // Executar cleanup
        r.deleteBucket(ctx, s3Client, bucket)

        // Remover finalizer
        controllerutil.RemoveFinalizer(bucket, s3BucketFinalizer)
        r.Update(ctx, bucket)
}
}
```

### 3. Status Conditions

Seguindo o padrão Kubernetes de conditions para reportar estado.

**Estrutura**:
```go
type Condition struct {
Type               string      // "Ready"
Status             string      // "True", "False", "Unknown"
LastTransitionTime metav1.Time
Reason             string      // "BucketReady", "CreationFailed"
Message            string      // Descrição detalhada
}
```

### 4. Provider Pattern

Separação de credenciais (AWSProvider) dos recursos (S3Bucket, RDS, etc).

**Vantagens**:
- Reutilização de credenciais entre múltiplos recursos
- Validação centralizada de autenticação
- Suporte para múltiplos providers (multi-conta, multi-região)
- Rotação de credenciais facilitada

### 5. Deletion Policies

Política flexível para controlar o que acontece com recursos AWS quando o CR é deletado.

**Opções**:
- **Delete**: Remove recurso AWS (padrão)
- **Retain**: Mantém recurso AWS, remove apenas o CR
- **Orphan**: Desacopla do operator mas mantém tudo

## Fluxo de Dados

### Criação de um S3Bucket

![S3Bucket Creation Flow](/img/diagrams/s3bucket-creation-flow.svg)

### Autenticação IRSA

![IRSA Authentication Flow](/img/diagrams/irsa-authentication-flow.svg)

## Estrutura de Arquivos

```
infra-operator/
│
├── api/v1alpha1/              # Definições de API (CRDs)
│   ├── groupversion_info.go  # Registro do grupo de API
│   ├── awsprovider_types.go  # Struct do CRD AWSProvider
│   ├── s3bucket_types.go     # Struct do CRD S3Bucket
│   ├── rdsinstance_types.go  # Struct do CRD RDS (controller pendente)
│   ├── ec2instance_types.go  # Struct do CRD EC2 (controller pendente)
│   └── sqsqueue_types.go     # Struct do CRD SQS (controller pendente)
│
├── controllers/               # Lógica de Reconciliação
│   ├── awsprovider_controller.go  # Valida credenciais AWS
│   └── s3bucket_controller.go     # Gerencia buckets S3
│
├── pkg/                       # Bibliotecas Compartilhadas
│   └── aws/
│       └── provider.go        # Funções auxiliares para config AWS
│
├── config/                    # Manifestos Kubernetes
│   ├── crd/bases/             # Manifestos YAML dos CRDs
│   ├── rbac/                  # RBAC (ServiceAccount, Role, Binding)
│   ├── manager/               # Deployment e Namespace
│   └── samples/               # CRs de exemplo
│
├── docs/                      # Documentação
│   ├── ARCHITECTURE.md        # Este arquivo
│   └── DEPLOYMENT_GUIDE.md    # Guia de deployment
│
├── main.go                    # Ponto de entrada do operator
├── Dockerfile                 # Build da imagem de container
├── Makefile                   # Automação de build
├── go.mod                     # Definição do módulo Go
├── CLAUDE.md                  # Documentação técnica completa
└── README.md                  # Documentação voltada ao usuário
```

## Decisões Arquiteturais

### Por que Go SDK v2 ao invés de ACK?

**Decisão**: Usar AWS SDK for Go v2 diretamente.

**Alternativa considerada**: AWS Controllers for Kubernetes (ACK)

**Razões**:

1. **Controle Total**:
   - Lógica de reconciliação customizada
   - Validações específicas do negócio
   - Tratamento de erros customizado

2. **Simplicidade no Deployment**:
   - Operator único para todos os serviços
   - Sem necessidade de instalar múltiplos controllers ACK
   - Menor overhead de recursos

3. **Aprendizado**:
   - Melhor entendimento dos padrões de operators
   - Conhecimento profundo das APIs AWS
   - Flexibilidade para implementar recursos customizados

4. **Dependências**:
   - Menos dependências externas
   - Updates de SDK mais simples
   - Sem dependência do roadmap ACK

**Trade-offs**:
- Mais código para manter
- Necessidade de implementar recursos manualmente (paginação, retries)
- Responsabilidade de acompanhar mudanças nas APIs AWS

### Por que Provider Pattern?

**Decisão**: Separar credenciais (AWSProvider) dos recursos (S3, RDS, etc).

**Razões**:

1. **Reuso**: Um AWSProvider pode ser usado por múltiplos recursos
2. **Segurança**: Credenciais isoladas em recurso específico
3. **Multi-tenancy**: Diferentes namespaces podem ter diferentes providers
4. **Multi-conta**: Suporta acesso a múltiplas contas AWS
5. **Rotação**: Facilita rotação de credenciais

**Alternativa**: Credenciais diretas em cada recurso (muito repetitivo e inseguro)

### Por que Deletion Policies?

**Decisão**: Permitir que usuários escolham o que acontece ao deletar um CR.

**Razões**:

1. **Segurança**: Evitar deleção acidental de dados importantes
2. **Flexibilidade**: Diferentes casos de uso (dev vs prod)
3. **Compliance**: Algumas regulamentações exigem retenção de dados
4. **Migração**: Facilita migração para outras ferramentas

**Políticas implementadas**:
- `Delete`: Remove recurso AWS (padrão para ambientes efêmeros)
- `Retain`: Mantém recurso AWS (padrão para produção)
- `Orphan`: Desacopla mas mantém tudo

## Concorrência e Thread Safety

### Controller Runtime

Controller-runtime gerencia concorrência automaticamente:

- **Work Queue**: Serializa eventos de reconciliação
- **Max Concurrent Reconciles**: Configurável (padrão: 1)
- **Retry Backoff**: Backoff exponencial para erros

### Estado Compartilhado

**Sem estado compartilhado entre reconciliações**:
- Cada reconciliação busca estado atual do Kubernetes
- Sem cache compartilhado entre reconciles
- Estado persistido apenas nos CRs

### Rate Limiting

**AWS SDK**:
- Implementa retry automático com backoff exponencial
- Respeita limites de taxa da AWS
- Usa context.Context para timeout

**Kubernetes Client**:
- Rate limiting configurável
- Padrão: 5 QPS, burst 10

## Performance e Escalabilidade

### Otimizações Implementadas

1. **Smart Requeue**:
   - Recursos ready: requeue a cada 5 minutos
   - Recursos com erro: requeue a cada 1 minuto
   - Mudanças imediatas: requeue imediato

2. **Minimizar Chamadas de API**:
   - HeadBucket antes de GetBucket (mais barato)
   - Aplicar configurações apenas se mudaram
   - Batch updates quando possível

3. **Status Subresource**:
   - Updates de status não triggam nova reconciliação
   - Reduz reconciliações desnecessárias

### Limites Atuais

1. **Região Única por Provider**: Um AWSProvider = uma região
   - **Workaround**: Criar múltiplos AWSProviders

2. **Sem Paginação**: Não lista grandes quantidades de recursos
   - **Impacto**: Limitado para operators que gerenciam recursos

3. **Processamento Sequencial**: Um recurso por vez
   - **Impacto**: Escalabilidade limitada para muitos recursos

### Melhorias Futuras

1. **Metrics**: Métricas Prometheus para observabilidade
2. **Webhooks**: Validação e defaults em admission
3. **Caching**: Cache de informações AWS para reduzir chamadas de API
4. **Parallel Reconciliation**: MaxConcurrentReconciles > 1
5. **Health Checks**: Health checks detalhados para recursos AWS

## Segurança

### Princípios Implementados

1. **Least Privilege**:
   - RBAC mínimo necessário
   - Políticas IAM específicas por serviço
   - Recursos namespaced quando possível

2. **Gerenciamento de Secrets**:
   - Credenciais em Kubernetes Secrets
   - IRSA preferido (sem credenciais de longa duração)
   - Nunca logar credenciais

3. **Container Security**:
   - Usuário não-root (UID 65532)
   - Root filesystem somente-leitura
   - Sem escalação privilegiada
   - Imagem base distroless

4. **Network Security**:
   - HTTPS para APIs AWS
   - Validação de certificados
   - Sem exposição de credenciais no status

### Modelo de Ameaças

**Ameaças consideradas**:
1. **Vazamento de Credenciais**: Mitigado com IRSA e Secrets
2. **Acesso Não Autorizado**: Mitigado com RBAC
3. **Escalação de Privilégio**: Mitigado com security context
4. **Ataques MITM**: Mitigado com HTTPS

## Observabilidade

### Logging

**Logging estruturado** com controller-runtime:
- Contexto automático (namespace, name, kind)
- Níveis de log: info, error, debug
- Formato JSON (opcional)

**Exemplo**:
```go
logger := log.FromContext(ctx)
logger.Info("Creating S3 bucket", "bucket", bucketName)
logger.Error(err, "Failed to create bucket")
```

### Relatório de Status

**Todos os CRs expõem**:
- `status.ready`: Booleano indicando se o recurso está pronto
- `status.conditions`: Array de condições detalhadas
- `status.lastSyncTime`: Timestamp da última reconciliação

### Health Checks

**Endpoints disponíveis**:
- `/healthz`: Liveness probe
- `/readyz`: Readiness probe
- `/metrics`: Métricas Prometheus (futuro)

## Estratégia de Testes

### Níveis de Teste

1. **Testes Unitários** (futuro):
   - Teste de lógica dos controllers
   - Mock do AWS SDK
   - Mock do cliente Kubernetes

2. **Testes de Integração** (futuro):
   - Teste com envtest (fake API server)
   - Teste de reconciliação completa
   - Sem dependência real de AWS

3. **Testes E2E** (futuro):
   - Teste em cluster real
   - Teste com AWS real
   - Validação de recursos criados

### Metas de Cobertura de Testes

- **Controllers**: > 80% de cobertura
- **AWS helpers**: > 90% de cobertura
- **E2E**: Happy paths e principais casos de erro

## Conclusão

O **infra-operator** foi arquitetado seguindo padrões estabelecidos pela comunidade Kubernetes, com foco em:

- **Simplicidade**: Fácil de entender e manter
- **Extensibilidade**: Fácil de adicionar novos recursos AWS
- **Segurança**: IRSA, RBAC, least privilege
- **Confiabilidade**: Finalizers, deletion policies, idempotência
- **Observabilidade**: Status conditions, logging estruturado

A arquitetura permite crescimento futuro com adição de novos serviços AWS, implementação de webhooks e melhorias de performance, mantendo uma fundação sólida e bem estruturada.
