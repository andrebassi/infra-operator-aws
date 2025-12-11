# Infra Operator

**Gerencie Infraestrutura AWS diretamente do Kubernetes**

Um operador Kubernetes que permite gerenciar recursos AWS de forma declarativa usando Custom Resources (CRs). Provisione e gerencie VPCs, Subnets, S3, RDS, EC2, SQS e mais usando ferramentas nativas do Kubernetes.

## 🚀 Features

- **Declarativo**: Defina recursos AWS como manifests Kubernetes
- **GitOps Ready**: Integração com ArgoCD, Flux e outras ferramentas GitOps
- **Múltiplos Métodos de Autenticação**: IRSA, credenciais estáticas, AssumeRole
- **Ciclo de Vida Completo**: Criação, atualização e deleção controlada
- **Production Ready**: Finalizers, status conditions, validação, RBAC
- **Clean Architecture**: Código testável, manutenível e extensível

## 📦 Serviços AWS Suportados (18 Total)

### ✅ Production Ready (LocalStack Community)

| Categoria | Serviços |
|-----------|----------|
| **Networking** | VPC, Subnet, Internet Gateway, NAT Gateway |
| **Storage** | S3 Bucket |
| **Database** | DynamoDB Table |
| **Compute** | EC2 Instance, Lambda Function |
| **Messaging** | SQS Queue, SNS Topic |
| **Security** | IAM Role, Secrets Manager, KMS Key |

### ⚠️ Requer LocalStack Pro ou AWS Real

| Categoria | Serviços |
|-----------|----------|
| **Database** | RDS Instance |
| **Container** | ECR Repository |
| **Caching** | ElastiCache Cluster |

## 🎯 Quick Start

### Pré-requisitos

- Kubernetes 1.28+
- kubectl configurado
- Conta AWS com permissões IAM (ou LocalStack para desenvolvimento)

### Instalação

```bash
# 1. Instalar CRDs
kubectl apply -f config/crd/bases/

# 2. Deploy do operador
kubectl apply -f config/manager/namespace.yaml
kubectl apply -f config/rbac/
kubectl apply -f config/manager/deployment.yaml

# 3. Verificar instalação
kubectl get pods -n infra-operator-system
```

Ou use o Makefile:

```bash
make install-complete
```

### Exemplo: Criar VPC e Subnet

```yaml
# 1. AWS Provider (Credenciais)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: my-aws
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator-role

---
# 2. VPC
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
spec:
  providerRef:
    name: my-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: production-vpc

---
# 3. Subnet
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-subnet-1a
spec:
  providerRef:
    name: my-aws
  vpcID: vpc-xxx  # Auto-preenchido pelo status da VPC
  cidrBlock: "10.0.1.0/24"
  availabilityZone: us-east-1a
  mapPublicIpOnLaunch: true
```

```bash
kubectl apply -f infrastructure.yaml

# Verificar status
kubectl get vpc,subnet
kubectl describe vpc production-vpc
```

## 📚 Documentação

### Início Rápido
- **[Quick Start](docs/QUICKSTART.md)** - Tutorial de 5 minutos
- **[Development](docs/DEVELOPMENT.md)** - Desenvolvimento local com LocalStack

### Documentação por Serviço

Consulte a documentação completa de cada serviço:

**Networking:**
- [VPC](docs/services/networking/vpc.mdx) - Virtual Private Cloud
- [Subnet](docs/services/networking/subnet.mdx) - Sub-redes
- [Internet Gateway](docs/services/networking/internet-gateway.mdx) - Gateway de internet
- [NAT Gateway](docs/services/networking/nat-gateway.mdx) - NAT para sub-redes privadas

**Storage:**
- [S3 Bucket](docs/services/storage/s3.mdx) - Object storage

**Database:**
- [DynamoDB](docs/services/database/dynamodb.mdx) - NoSQL database
- [RDS](docs/services/database/rds.mdx) - Relational database ⚠️ Pro

**Compute:**
- [EC2](docs/services/compute/ec2.mdx) - Virtual machines
- [Lambda](docs/services/compute/lambda.mdx) - Serverless functions

**Messaging:**
- [SQS](docs/services/messaging/sqs.mdx) - Message queues
- [SNS](docs/services/messaging/sns.mdx) - Pub/Sub notifications

**Security:**
- [IAM Role](docs/services/security/iam.mdx) - Identity and access
- [Secrets Manager](docs/services/security/secrets-manager.mdx) - Secrets storage
- [KMS](docs/services/security/kms.mdx) - Encryption keys

**Container:**
- [ECR](docs/services/container/ecr.mdx) - Container registry ⚠️ Pro

**Caching:**
- [ElastiCache](docs/services/caching/elasticache.mdx) - In-memory cache ⚠️ Pro

### Guias Completos
- **[Services Guide](docs/SERVICES_GUIDE.md)** - Documentação all-in-one de todos os serviços
- **[Architecture](docs/ARCHITECTURE.md)** - Arquitetura do sistema
- **[Clean Architecture](docs/CLEAN_ARCHITECTURE.md)** - Implementação Clean Architecture
- **[Deployment](docs/DEPLOYMENT_GUIDE.md)** - Deploy em produção

### Documentação Interativa (Mintlify)

```bash
npm i -g mintlify
cd docs
mintlify dev
# Acesse: http://localhost:3000
```

## 🏗️ Arquitetura

O Infra Operator segue princípios de **Clean Architecture**:

```
┌─────────────────────────────────┐
│   Controllers (Kubernetes)       │
│   - VPCReconciler               │
│   - SubnetReconciler            │
└────────────┬────────────────────┘
             │
┌────────────▼────────────────────┐
│   Use Cases (Business Logic)    │
│   - CreateVPC, UpdateVPC        │
└────────────┬────────────────────┘
             │
┌────────────▼────────────────────┐
│   Ports (Interfaces)            │
│   - AWSClientPort               │
└────────────┬────────────────────┘
             │
┌────────────▼────────────────────┐
│   Adapters (AWS SDK)            │
│   - EC2Adapter, S3Adapter       │
└─────────────────────────────────┘
```

**Benefícios:**
- ✅ Testabilidade (domain isolado)
- ✅ Manutenibilidade (separação de responsabilidades)
- ✅ Flexibilidade (fácil adicionar novos serviços)
- ✅ Independência (sem acoplamento com frameworks)

## 🔐 Autenticação

### IRSA (Recomendado para EKS)

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-prod
spec:
  region: us-east-1
  roleARN: arn:aws:iam::ACCOUNT_ID:role/infra-operator-role
```

### Credenciais Estáticas (Desenvolvimento)

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-dev
spec:
  region: us-west-2
  accessKeyIDRef:
    name: aws-creds
    key: access-key-id
  secretAccessKeyRef:
    name: aws-creds
    key: secret-access-key
```

## 🛠️ Desenvolvimento

### Desenvolvimento Local com LocalStack

```bash
# Setup completo (LocalStack + tools)
task setup

# Rodar operador localmente
task run:local

# Rodar testes
task test:all
```

### Build e Deploy

```bash
# Build
task build              # Build binário
task docker:build       # Build imagem Docker

# Deploy
task k8s:deploy        # Deploy no cluster
task samples:apply     # Deploy recursos de exemplo

# Logs
task k8s:logs          # Ver logs do operador
```

Veja mais detalhes em [Development Guide](docs/DEVELOPMENT.md).

## 🗺️ Roadmap

### ✅ Concluído
- [x] Core operator framework com Clean Architecture
- [x] 14 serviços AWS funcionando no LocalStack Community
- [x] 3 serviços AWS para LocalStack Pro/AWS Real
- [x] Documentação completa (Markdown + Mintlify)
- [x] Testes de integração com LocalStack

### 🚧 Em Progresso
- [ ] Validation webhooks
- [ ] Prometheus metrics
- [ ] Helm chart
- [ ] E2E test suite

### 📋 Planejado
- [ ] Mais serviços AWS (CloudFront, Route53, ALB, etc.)
- [ ] Multi-region resource management
- [ ] Cost estimation em status
- [ ] Drift detection e reconciliação

## 🤝 Contribuindo

Contribuições são bem-vindas! Áreas para melhorias:

1. **Novos Serviços AWS**: CloudFront, Route53, ALB, etc.
2. **Webhooks**: Admission webhooks para validação
3. **Metrics**: Export de métricas Prometheus
4. **Testes**: Mais testes unitários e E2E
5. **Documentação**: Mais exemplos e casos de uso

## 📄 Licença

MIT License - Veja arquivo LICENSE

## 🙏 Agradecimentos

Construído com:
- [Kubebuilder](https://book.kubebuilder.io/) - Framework para operadores
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Controller library
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/) - Cliente AWS API

---

**Versão:** v1.0.0
**Última Atualização:** 2025-11-22

Para suporte, issues ou contribuições, consulte a [documentação completa](docs/).
