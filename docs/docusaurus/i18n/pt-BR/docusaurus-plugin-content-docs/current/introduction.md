---
title: 'Introdução'
description: 'Gerencie infraestrutura AWS diretamente do Kubernetes com o Infra Operator'
slug: /
sidebar_position: 1
---

Gerencie recursos AWS diretamente do Kubernetes usando Custom Resources e GitOps.

## O que é o Infra Operator?

**Infra Operator** é um operador Kubernetes que permite provisionar e gerenciar recursos AWS usando Custom Resource Definitions (CRDs). Em vez de usar ferramentas separadas como Terraform ou CloudFormation, você pode gerenciar sua infraestrutura AWS usando `kubectl` e ferramentas GitOps como ArgoCD.

### Principais Benefícios

Gerencie infraestrutura junto com aplicações usando Git como fonte da verdade

Use kubectl, Helm e ferramentas familiares do ecossistema Kubernetes

Código bem organizado, testável (100% de cobertura) e manutenível seguindo padrões de arquitetura

Suporte completo para rede, computação, armazenamento, banco de dados, mensageria, CDN, segurança e mais

## Serviços Suportados (26 no Total)

### Rede (9 serviços)

| Serviço | Descrição |
|---------|-----------|
| **VPC** | Virtual Private Cloud |
| **Subnet** | Subnets da VPC |
| **Internet Gateway** | Acesso à internet para VPC |
| **NAT Gateway** | Internet de saída para subnets privadas |
| **Security Group** | Regras de firewall |
| **Route Table** | Roteamento de rede |
| **ALB** | Application Load Balancer (Camada 7) |
| **NLB** | Network Load Balancer (Camada 4) |
| **Elastic IP** | Endereços IP públicos estáticos |

### Computação (3 serviços)

| Serviço | Descrição |
|---------|-----------|
| **EC2 Instance** | Máquinas virtuais |
| **Lambda** | Funções serverless |
| **EKS** | Clusters Kubernetes |

### Armazenamento & Banco de Dados (3 serviços)

| Serviço | Descrição |
|---------|-----------|
| **S3 Bucket** | Armazenamento de objetos |
| **RDS Instance** | Bancos de dados relacionais (PostgreSQL, MySQL, etc.) |
| **DynamoDB Table** | Banco de dados NoSQL |

### Mensageria (2 serviços)

| Serviço | Descrição |
|---------|-----------|
| **SQS Queue** | Filas de mensagens |
| **SNS Topic** | Notificações Pub/Sub |

### API & CDN (2 serviços)

| Serviço | Descrição |
|---------|-----------|
| **API Gateway** | APIs REST, HTTP, WebSocket |
| **CloudFront** | Rede de Distribuição de Conteúdo (CDN) |

### Segurança (4 serviços)

| Serviço | Descrição |
|---------|-----------|
| **IAM Role** | Gerenciamento de identidade e acesso |
| **Secrets Manager** | Armazenamento de segredos |
| **KMS Key** | Chaves de criptografia |
| **ACM Certificate** | Certificados SSL/TLS |

### Containers (2 serviços)

| Serviço | Descrição |
|---------|-----------|
| **ECR Repository** | Registro de containers |
| **ECS Cluster** | Orquestração de containers |

### Cache (1 serviço)

| Serviço | Descrição |
|---------|-----------|
| **ElastiCache** | Cache em memória (Redis, Memcached) |

## Exemplo Rápido

**Exemplo:**

```yaml
---
# Criar VPC
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
spec:
  providerRef:
    name: aws-provider
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true

---
# Criar Subnet Pública
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
metadata:
  name: public-subnet
spec:
  providerRef:
    name: aws-provider
  vpcID: vpc-xxx  # Será preenchido automaticamente
  cidrBlock: "10.0.1.0/24"
  mapPublicIpOnLaunch: true

---
# Criar Bucket S3
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: app-data
spec:
  providerRef:
    name: aws-provider
  bucketName: myapp-production-data
  versioning:
    enabled: true
  encryption:
    algorithm: AES256
```

**Comando:**

```bash
kubectl apply -f infrastructure.yaml
```

## Arquitetura

### Visão Geral do Sistema

O Infra Operator segue a arquitetura do padrão controller do Kubernetes:

**Como Funciona:**

1. **GitOps / kubectl** - Cria Custom Resources (CRs) no Kubernetes
2. **Controllers** - Detectam mudanças nos CRs e reconciliam o estado
3. **AWS SDK** - Controllers usam o AWS SDK para provisionar recursos
4. **Atualização de Status** - Estado do recurso AWS é refletido no CR

**Componentes Principais:**

- **26 Controllers**: Um para cada serviço AWS (VPC, S3, EC2, Lambda, etc.)
- **26 CRDs**: Custom Resource Definitions para cada tipo de recurso
- **AWS SDK v2**: Comunicação com as APIs AWS
- **Loop de Reconciliação**: Garante que estado desejado = estado atual

### Clean Architecture

Implementação seguindo princípios de **Clean Architecture** para código testável (100% de cobertura), manutenível e desacoplado:

**Camadas da Arquitetura:**

**1. Camada de Domínio (Core)**
- Modelos de negócio puros (VPC, S3, EC2, etc.)
- Regras de validação
- Sem dependências externas
- 100% de cobertura de testes

**2. Camada de Casos de Uso**
- Lógica de aplicação
- Criar, Atualizar, Deletar, GetStatus
- Orquestração de domínio
- Interface com Ports

**3. Camada de Ports (Interfaces)**
- Interfaces de repositório
- Interfaces de Cloud Provider
- Princípio de inversão de dependência
- Contratos abstratos

**4. Camada de Adapters**
- Implementações do AWS SDK
- Cliente Kubernetes
- Implementações concretas dos Ports
- Comunicação com sistemas externos

**5. Camada de Controllers**
- Loops de reconciliação
- Integração com API Kubernetes
- Tratamento de eventos
- Mapeamento de CR para Domínio

**Benefícios:**
- Testabilidade: 100% de cobertura no domínio
- Manutenibilidade: Clara separação de responsabilidades
- Flexibilidade: Fácil adicionar novos serviços
- Independência: Core desacoplado de frameworks

## Próximos Passos

- [Instalação](/installation) - Como instalar o Infra Operator
- [Início Rápido](/quickstart) - Primeiros passos
- [Serviços AWS](/services/networking/vpc) - Documentação dos serviços
- [Referência da API](/api-reference/overview) - Referência completa da API
