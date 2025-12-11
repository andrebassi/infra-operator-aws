---
title: 'AWSProvider'
description: 'Configurar credenciais e configurações AWS para o Infra Operator'
sidebar_position: 2
---

# AWSProvider

O recurso `AWSProvider` configura credenciais e configurações AWS que outros recursos usam para interagir com a AWS.

## Visão Geral

Todo recurso AWS gerenciado pelo Infra Operator deve referenciar um AWSProvider. O provider lida com:

- Credenciais AWS (estáticas ou IRSA)
- Configuração de região
- Endpoint customizado (para testes com LocalStack)
- Tags padrão aplicadas a todos os recursos

## Início Rápido

### Usando IRSA (Recomendado para Produção)

**Exemplo:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-production
  namespace: infra-operator
spec:
  region: us-east-1
  # IRSA: Sem credenciais necessárias, usa anotações do ServiceAccount
  defaultTags:
    ManagedBy: infra-operator
    Environment: production
```

### Usando Credenciais Estáticas

**Exemplo:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: infra-operator
type: Opaque
stringData:
  AWS_ACCESS_KEY_ID: "AKIAIOSFODNN7EXAMPLE"
  AWS_SECRET_ACCESS_KEY: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-production
  namespace: infra-operator
spec:
  region: us-east-1
  credentialsSecret:
    name: aws-credentials
    namespace: infra-operator
  defaultTags:
    ManagedBy: infra-operator
```

### Usando LocalStack (Desenvolvimento/Testes)

**Exemplo:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: localstack
  namespace: infra-operator
spec:
  region: us-east-1
  endpoint: http://localstack.default.svc.cluster.local:4566
  credentialsSecret:
    name: aws-credentials
    namespace: infra-operator
  defaultTags:
    ManagedBy: infra-operator
    Environment: localstack
```

## Especificação

### Campos Obrigatórios

| Campo | Tipo | Descrição |
|-------|------|-------------|
| `spec.region` | string | Região AWS (ex: `us-east-1`, `eu-west-1`) |

### Campos Opcionais

| Campo | Tipo | Padrão | Descrição |
|-------|------|---------|-------------|
| `spec.credentialsSecret` | object | - | Referência a Secret com credenciais AWS |
| `spec.endpoint` | string | - | Endpoint AWS customizado (para LocalStack) |
| `spec.roleARN` | string | - | ARN do Role IAM para acesso cross-account |
| `spec.defaultTags` | object | - | Tags aplicadas a todos os recursos |

### CredentialsSecret

**Exemplo:**

```yaml
spec:
  credentialsSecret:
    name: aws-credentials      # Nome do Secret
    namespace: infra-operator  # Namespace do Secret
```

O Secret deve conter:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

Opcionalmente:
- `AWS_SESSION_TOKEN` (para credenciais temporárias)

## Campos de Status

| Campo | Tipo | Descrição |
|-------|------|-------------|
| `status.ready` | boolean | Provider está configurado e pronto |
| `status.accountID` | string | ID da Conta AWS |
| `status.region` | string | Região AWS configurada |
| `status.lastSyncTime` | string | Última chamada de API AWS bem-sucedida |

## Configuração IRSA

Para deployments de produção no EKS, use IRSA (IAM Roles for Service Accounts):

### 1. Criar Role IAM

**Comando:**

```bash
# Obter Provedor OIDC
export CLUSTER_NAME=my-cluster
export AWS_REGION=us-east-1
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

OIDC_PROVIDER=$(aws eks describe-cluster \
  --name $CLUSTER_NAME \
  --region $AWS_REGION \
  --query "cluster.identity.oidc.issuer" \
  --output text | sed -e "s/^https:\/\///")

# Criar trust policy
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
          "${OIDC_PROVIDER}:sub": "system:serviceaccount:infra-operator:infra-operator",
          "${OIDC_PROVIDER}:aud": "sts.amazonaws.com"
        }
      }
}
  ]
}
EOF

# Criar role
aws iam create-role \
  --role-name infra-operator-role \
  --assume-role-policy-document file://trust-policy.json
```

### 2. Anexar Políticas

**Comando:**

```bash
# Anexar políticas gerenciadas ou criar customizadas
aws iam attach-role-policy \
  --role-name infra-operator-role \
  --policy-arn arn:aws:iam::aws:policy/AmazonVPCFullAccess

aws iam attach-role-policy \
  --role-name infra-operator-role \
  --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess

# Adicionar mais políticas conforme necessário para seus recursos
```

### 3. Anotar ServiceAccount

**Comando:**

```bash
kubectl annotate serviceaccount infra-operator \
  -n infra-operator \
  eks.amazonaws.com/role-arn=arn:aws:iam::${AWS_ACCOUNT_ID}:role/infra-operator-role
```

## Configuração Multi-Conta

Para gerenciar recursos em múltiplas contas AWS:

**Exemplo:**

```yaml
# Conta A (Primária)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: account-a
spec:
  region: us-east-1
  credentialsSecret:
    name: account-a-credentials
---
# Conta B (Secundária)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: account-b
spec:
  region: us-east-1
  roleARN: arn:aws:iam::222222222222:role/infra-operator-cross-account
  credentialsSecret:
    name: account-a-credentials  # Usar credenciais primárias para assumir role
```

## Melhores Práticas

:::note Melhores Práticas

- **Use IRSA em produção** — Nunca use credenciais estáticas em produção, aproveite IAM Roles for Service Accounts
- **Aplique políticas IAM de menor privilégio** — Conceda apenas as permissões que o operator realmente precisa
- **Rotacione credenciais regularmente** — Se usar credenciais estáticas, implemente rotação regular
- **Use providers separados por ambiente** — Mantenha dev/staging/prod isolados com diferentes providers
- **Um provider por conta/região AWS** — Use nomes descritivos como aws-prod-us-east-1, aws-dev-eu-west-1
- **Aplique tags padrão consistentes** — Configure tags ManagedBy, Environment no nível do provider para todos os recursos

:::

## Troubleshooting

### Provider Não Ready

**Comando:**

```bash
# Verificar status do provider
kubectl describe awsprovider aws-production

# Verificar logs do operator
kubectl logs -n infra-operator deploy/infra-operator --tail=100

# Verificar se Secret de credenciais existe
kubectl get secret aws-credentials -n infra-operator
```

### Credenciais Inválidas

**Comando:**

```bash
# Testar credenciais manualmente
export AWS_ACCESS_KEY_ID=$(kubectl get secret aws-credentials -n infra-operator -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 -d)
export AWS_SECRET_ACCESS_KEY=$(kubectl get secret aws-credentials -n infra-operator -o jsonpath='{.data.AWS_SECRET_ACCESS_KEY}' | base64 -d)

aws sts get-caller-identity
```

### IRSA Não Está Funcionando

**Comando:**

```bash
# Verificar anotações do ServiceAccount
kubectl get sa infra-operator -n infra-operator -o yaml

# Verificar se provedor OIDC está configurado
aws eks describe-cluster --name $CLUSTER_NAME --query "cluster.identity.oidc"

# Verificar trust policy do role IAM
aws iam get-role --role-name infra-operator-role --query "Role.AssumeRolePolicyDocument"
```
