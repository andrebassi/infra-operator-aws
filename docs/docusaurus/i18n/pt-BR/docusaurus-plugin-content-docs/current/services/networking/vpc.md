---
title: 'VPC - Virtual Private Cloud'
description: 'Crie redes virtuais isoladas na AWS'
sidebar_position: 9
---

Crie redes virtuais isoladas na AWS para hospedar seus recursos com segurança.

## Pré-requisito: Configuração do AWSProvider

Antes de criar qualquer recurso AWS, você precisa configurar um **AWSProvider** que gerencia as credenciais e autenticação com a AWS.

**IRSA:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: production-aws
  namespace: default
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator-role
  defaultTags:
    managed-by: infra-operator
    environment: production
```

**Credenciais Estáticas:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: default
type: Opaque
stringData:
  access-key-id: test
  secret-access-key: test
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: localstack
  namespace: default
spec:
  region: us-east-1
  accessKeyIDRef:
    name: aws-credentials
    key: access-key-id
  secretAccessKeyRef:
    name: aws-credentials
    key: secret-access-key
  defaultTags:
    managed-by: infra-operator
    environment: test
```

**Verificar Status:**

```bash
kubectl get awsprovider
kubectl describe awsprovider production-aws
```
:::warning

Para produção, sempre use **IRSA** (IAM Roles for Service Accounts) ao invés de credenciais estáticas.

:::

### Criar Role IAM para IRSA

Para usar IRSA em produção, você precisa criar uma Role IAM com as permissões necessárias:

**Trust Policy (trust-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE:sub": "system:serviceaccount:infra-operator-system:infra-operator-controller-manager",
          "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

**IAM Policy - VPC (vpc-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateVpc",
        "ec2:DeleteVpc",
        "ec2:DescribeVpcs",
        "ec2:ModifyVpcAttribute",
        "ec2:CreateTags",
        "ec2:DeleteTags",
        "ec2:DescribeTags"
      ],
      "Resource": "*"
    }
  ]
}
```

**Criar Role com AWS CLI:**

```bash
# 1. Obter OIDC Provider do cluster EKS
export CLUSTER_NAME=my-cluster
export AWS_REGION=us-east-1
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

OIDC_PROVIDER=$(aws eks describe-cluster \
  --name $CLUSTER_NAME \
  --region $AWS_REGION \
  --query "cluster.identity.oidc.issuer" \
  --output text | sed -e "s/^https:\/\///")

# 2. Atualizar trust-policy.json com valores corretos
cat > trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${OIDC_PROVIDER}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${OIDC_PROVIDER}:sub": "system:serviceaccount:infra-operator-system:infra-operator-controller-manager",
          "${OIDC_PROVIDER}:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
EOF

# 3. Criar IAM Role
aws iam create-role \
  --role-name infra-operator-vpc-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator VPC management"

# 4. Criar e anexar policy
aws iam put-role-policy \
  --role-name infra-operator-vpc-role \
  --policy-name VPCManagement \
  --policy-document file://vpc-policy.json

# 5. Obter Role ARN
aws iam get-role \
  --role-name infra-operator-vpc-role \
  --query 'Role.Arn' \
  --output text
```

**Anotar ServiceAccount do Operator:**

```bash
# Adicionar annotation ao ServiceAccount do operator
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-vpc-role
```
:::note

Substitua `123456789012` pelo seu AWS Account ID e `EXAMPLED539D4633E53DE1B71EXAMPLE` pelo seu ID do provedor OIDC.

:::

## Visão Geral

Uma Virtual Private Cloud (VPC) é uma seção logicamente isolada da nuvem AWS onde você pode lançar recursos AWS em uma rede virtual que você define. Com VPC, você tem controle completo sobre seu ambiente de rede virtual, incluindo:

- Seleção de faixas de endereços IP
- Criação de sub-redes
- Configuração de tabelas de rotas
- Configuração de gateways de rede

## Início Rápido

**VPC Básica:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: e2e-test-vpc
  namespace: default
spec:
  providerRef:
    name: localstack
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: e2e-test-vpc
    Environment: testing
    ManagedBy: infra-operator
  deletionPolicy: Delete
```

**VPC de Produção:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
  namespace: default
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: production-vpc
    Environment: production
    Team: platform
    CostCenter: engineering
  deletionPolicy: Retain
```

**Aplicar:**

```bash
kubectl apply -f vpc.yaml
```

**Verificar Status:**

```bash
kubectl get vpc
kubectl describe vpc e2e-test-vpc
```
## Referência de Configuração

### Campos Obrigatórios

Referência ao recurso AWSProvider

  Nome do recurso AWSProvider

Bloco CIDR IPv4 para a VPC (ex: "10.0.0.0/16")

  **Faixas válidas:**
  - 10.0.0.0 - 10.255.255.255 (prefixo 10/8)
  - 172.16.0.0 - 172.31.255.255 (prefixo 172.16/12)
  - 192.168.0.0 - 192.168.255.255 (prefixo 192.168/16)

  **Netmask permitida:** /16 a /28

### Campos Opcionais

Habilita resolução DNS na VPC. Quando habilitado, instâncias na VPC podem resolver nomes de host DNS.

Habilita nomes de host DNS na VPC. Instâncias recebem nomes de host DNS públicos que correspondem aos seus endereços IP públicos.

  :::note

Requer `enableDnsSupport: true`
:::


Opção de tenancy para instâncias lançadas na VPC

  **Opções:**
  - `default`: Instâncias executam em hardware compartilhado
  - `dedicated`: Instâncias executam em hardware single-tenant (custo adicional)

Pares chave-valor para marcar a VPC

  **Exemplo:**

  ```yaml
  tags:
    Name: production-vpc
    Environment: production
    Team: platform
  ```

O que acontece com a VPC quando o CR é excluído

  **Opções:**
  - `Delete`: VPC é excluída da AWS
  - `Retain`: VPC permanece na AWS mas não gerenciada
  - `Orphan`: VPC permanece mas propriedade do CR é removida

## Campos de Status

Após a VPC ser criada, os seguintes campos de status são populados:

Identificador AWS da VPC (ex: `vpc-f3ea9b1b36fce09cd`)

Bloco CIDR atribuído à VPC

Estado atual da VPC
  - `pending`: VPC está sendo criada
  - `available`: VPC está pronta para uso

`true` quando a VPC está disponível e pronta para uso

Timestamp da última sincronização com a AWS

## Exemplos

### VPC de Produção

VPC de alta disponibilidade para cargas de trabalho de produção:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: production-vpc
  namespace: default
spec:
  providerRef:
    name: production-aws

  # CIDR grande para muitas sub-redes
  cidrBlock: "10.0.0.0/16"

  # Habilitar DNS para service discovery
  enableDnsSupport: true
  enableDnsHostnames: true

  # Hardware compartilhado (econômico)
  instanceTenancy: default

  tags:
    Name: production-vpc
    Environment: production
    ManagedBy: infra-operator
    CostCenter: engineering

  # Reter VPC se CR for excluído
  deletionPolicy: Retain
```

### VPC de Desenvolvimento

VPC menor para ambiente de desenvolvimento:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: dev-vpc
  namespace: default
spec:
  providerRef:
    name: localstack

  # CIDR menor para dev
  cidrBlock: "172.16.0.0/16"

  enableDnsSupport: true
  enableDnsHostnames: true

  tags:
    Name: development-vpc
    Environment: development
    AutoShutdown: "true"

  # Excluir VPC na limpeza
  deletionPolicy: Delete
```

### VPCs Multi-Ambiente

VPCs separadas para cada ambiente:

**Produção:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: prod-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.0.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: production-vpc
    Environment: production
  deletionPolicy: Retain
```

**Staging:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: staging-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.1.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: staging-vpc
    Environment: staging
  deletionPolicy: Delete
```

**Desenvolvimento:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: dev-vpc
spec:
  providerRef:
    name: production-aws
  cidrBlock: "10.2.0.0/16"
  enableDnsSupport: true
  enableDnsHostnames: true
  tags:
    Name: development-vpc
    Environment: development
  deletionPolicy: Delete
```
## Verificação

### Verificar Status da VPC

**Comando:**

```bash
# Listar todas as VPCs
kubectl get vpcs

# Obter informações detalhadas da VPC
kubectl get vpc production-vpc -o yaml

# Acompanhar criação da VPC
kubectl get vpc production-vpc -w
```

### Verificar na AWS

**AWS CLI:**

```bash
# Listar VPCs
aws ec2 describe-vpcs --vpc-ids vpc-xxx

# Obter detalhes da VPC
aws ec2 describe-vpcs \
      --vpc-ids vpc-xxx \
      --query 'Vpcs[0]' \
      --output json
```

**LocalStack:**

```bash
# Para testes com LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

aws ec2 describe-vpcs \
      --vpc-ids vpc-xxx
```

### Saída Esperada

**Exemplo:**

```yaml
status:
  vpcID: vpc-f3ea9b1b36fce09cd
  cidrBlock: 10.0.0.0/16
  state: available
  ready: true
  lastSyncTime: "2025-11-22T20:18:08Z"
```

## Solução de Problemas

### VPC travada em estado pending

**Sintomas:** `state` da VPC é `pending` por mais de 2 minutos

**Causas comuns:**
1. Credenciais do AWSProvider inválidas
2. Problemas de conectividade de rede
3. Limite de taxa da API AWS

**Soluções:**
```bash
# Verificar status do AWSProvider
kubectl describe awsprovider production-aws

# Verificar logs do controller
kubectl logs -n infra-operator-system \
      deploy/infra-operator-controller-manager \
      --tail=100

# Verificar eventos da VPC
kubectl describe vpc production-vpc
```

### Bloco CIDR já existe

**Erro:** `"CIDR block 10.0.0.0/16 conflicts with existing VPC"`

**Causa:** Outra VPC com o mesmo CIDR existe na conta

**Soluções:**
- Use um bloco CIDR diferente
- Exclua a VPC conflitante
- Use VPC peering se conectividade for necessária

### Exclusão travada

**Sintomas:** Exclusão da VPC demora muito ou trava

**Causa:** Recursos ainda anexados à VPC (sub-redes, IGW, etc.)

**Soluções:**
```bash
# Verificar recursos anexados
aws ec2 describe-subnets --filters "Name=vpc-id,Values=vpc-xxx"
aws ec2 describe-internet-gateways --filters "Name=attachment.vpc-id,Values=vpc-xxx"

# Excluir recursos dependentes primeiro
kubectl delete subnet <subnet-name>
kubectl delete internetgateway <igw-name>

# Depois excluir a VPC
kubectl delete vpc production-vpc
```

### DNS não funcionando

**Sintomas:** Instâncias não conseguem resolver nomes de host DNS

**Soluções:**
1. Certifique-se de `enableDnsSupport: true`
2. Certifique-se de `enableDnsHostnames: true` para DNS público
3. Verifique opções DHCP da VPC

**Exemplo:**

```yaml
spec:
      enableDnsSupport: true
      enableDnsHostnames: true
```

## Boas Práticas

:::note Boas Práticas

- **Use /16 CIDR para produção** — 65.536 IPs fornece espaço para crescimento
- **Reserve espaço para crescimento futuro** — Planeje blocos CIDR para evitar sobreposições com VPCs pareadas
- **Habilite hostnames DNS** — Necessário para zonas hospedadas privadas e service discovery
- **Habilite suporte DNS** — Necessário para resolução DNS da VPC
- **Use padrões consistentes de CIDR** — Exemplo: 10.{env}.0.0/16 onde env=0 para prod, 1 para staging

:::

## Padrões de Arquitetura

### Arquitetura de VPC Única

Configuração típica com sub-redes públicas e privadas distribuídas em múltiplas zonas de disponibilidade para alta disponibilidade:

![Arquitetura de VPC Única](/img/diagrams/vpc-single-architecture.svg)

### Arquitetura Multi-VPC

Arquitetura com múltiplas VPCs isoladas por ambiente, conectadas via VPC Peering para comunicação segura:

![Arquitetura Multi-VPC](/img/diagrams/vpc-multi-architecture.svg)

## Recursos Relacionados

- [Subnet](/services/networking/subnet)

  - [Internet Gateway](/services/networking/internet-gateway)

  - [NAT Gateway](/services/networking/nat-gateway)

  - [Guia de Rede Multi-Camadas](/guides/multi-tier-network)
