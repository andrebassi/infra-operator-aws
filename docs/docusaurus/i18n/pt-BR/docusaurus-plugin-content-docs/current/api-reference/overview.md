---
title: 'Visão Geral da API Reference'
description: 'Referência completa para todos os CRDs e APIs do Infra Operator'
sidebar_position: 1
---

# API Reference

Documentação de referência completa para os Custom Resource Definitions (CRDs) do Infra Operator.

## Grupo de API

Todos os recursos do Infra Operator usam o seguinte grupo de API:

**Grupo de API:**

```
aws-infra-operator.runner.codes/v1alpha1
```

## Recursos Disponíveis

### Recursos Core

| Kind | Descrição | Status |
|------|-----------|--------|
| `AWSProvider` | Credenciais e configuração AWS | Estável |

### Recursos de Rede

| Kind | Descrição | Status |
|------|-----------|--------|
| `VPC` | Virtual Private Cloud | Estável |
| `Subnet` | Subnet da VPC | Estável |
| `InternetGateway` | Internet Gateway | Estável |
| `NATGateway` | NAT Gateway | Estável |
| `RouteTable` | Route Table | Estável |
| `SecurityGroup` | Security Group | Estável |
| `ElasticIP` | Endereço IP Elástico | Estável |
| `ALB` | Application Load Balancer | Estável |
| `NLB` | Network Load Balancer | Estável |

### Recursos de Computação

| Kind | Descrição | Status |
|------|-----------|--------|
| `EC2Instance` | Instância EC2 | Estável |
| `EKSCluster` | Cluster Kubernetes EKS | Estável |
| `ECSCluster` | Cluster de Containers ECS | Estável |
| `LambdaFunction` | Função Lambda | Estável |
| `ComputeStack` | Infraestrutura All-in-One | Estável |

### Recursos de Storage

| Kind | Descrição | Status |
|------|-----------|--------|
| `S3Bucket` | Bucket S3 | Estável |

### Recursos de Banco de Dados

| Kind | Descrição | Status |
|------|-----------|--------|
| `RDSInstance` | Instância de Banco de Dados RDS | Estável |
| `DynamoDBTable` | Tabela DynamoDB | Estável |
| `ElastiCacheCluster` | Cluster ElastiCache | Estável |

### Recursos de Containers

| Kind | Descrição | Status |
|------|-----------|--------|
| `ECRRepository` | Container Registry ECR | Estável |

### Recursos de Mensageria

| Kind | Descrição | Status |
|------|-----------|--------|
| `SQSQueue` | Fila SQS | Estável |
| `SNSTopic` | Tópico SNS | Estável |

### Recursos de Segurança

| Kind | Descrição | Status |
|------|-----------|--------|
| `IAMRole` | Role IAM | Estável |
| `KMSKey` | Chave de Criptografia KMS | Estável |
| `SecretsManagerSecret` | Secret do Secrets Manager | Estável |
| `Certificate` | Certificado ACM | Estável |
| `EC2KeyPair` | Par de Chaves SSH EC2 | Estável |

### Recursos de CDN & DNS

| Kind | Descrição | Status |
|------|-----------|--------|
| `CloudFront` | Distribuição CloudFront | Estável |
| `Route53HostedZone` | Zona Hospedada Route53 | Estável |
| `Route53RecordSet` | Registro DNS Route53 | Estável |

### Gerenciamento de API

| Kind | Descrição | Status |
|------|-----------|--------|
| `APIGateway` | API Gateway | Estável |

## Campos Comuns

### ProviderRef

Todos os recursos AWS requerem uma referência a um AWSProvider:

**Exemplo:**

```yaml
spec:
  providerRef:
    name: aws-production  # Nome do recurso AWSProvider
```

### Tags

A maioria dos recursos suporta tags AWS:

**Exemplo:**

```yaml
spec:
  tags:
    Environment: production
    Team: platform
    ManagedBy: infra-operator
```

### DeletionPolicy

Controla o que acontece quando o recurso Kubernetes é deletado:

**Exemplo:**

```yaml
spec:
  deletionPolicy: Delete  # Delete | Retain | Orphan
```

- `Delete`: Deleta o recurso AWS quando o CR é deletado (padrão)
- `Retain`: Mantém o recurso AWS mas remove do gerenciamento do operator
- `Orphan`: Mantém o recurso AWS e remove metadados de ownership

## Campos de Status

Todos os recursos expõem campos de status comuns:

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `ready` | boolean | Se o recurso está pronto para uso |
| `lastSyncTime` | string | Última sincronização bem-sucedida com AWS |
| `conditions` | array | Condições de status detalhadas |

## Exemplo

**Exemplo:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
  namespace: infra-operator
spec:
  providerRef:
    name: aws-production
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Environment: production
  deletionPolicy: Retain
status:
  vpcID: vpc-0123456789abcdef0
  cidrBlock: "10.0.0.0/16"
  state: available
  ready: true
  lastSyncTime: "2025-11-22T20:18:08Z"
```

## Próximos Passos

- [Configuração do AWSProvider](/api-reference/awsprovider)
- [Especificações dos CRDs](/api-reference/crds)
- [Recursos de Rede](/services/networking/vpc)
