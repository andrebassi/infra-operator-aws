# Guia de Deployment - Infra Operator

Este guia descreve cada passo necessário para fazer deploy do **infra-operator** em um cluster Kubernetes.

## Índice

1. [Pré-requisitos](#pré-requisitos)
2. [Construindo o Operator](#construindo-o-operator)
3. [Configuração IAM/IRSA](#configuração-iamirsa)
4. [Instalação no Cluster](#instalação-no-cluster)
5. [Verificação](#verificação)
6. [Primeiro Recurso](#primeiro-recurso)
7. [Troubleshooting](#troubleshooting)

---

## Pré-requisitos

### 1. Cluster Kubernetes

- **Versão**: 1.28 ou superior
- **Acesso**: `kubectl` configurado e funcionando
- **Tipo**: Qualquer distribuição (EKS, GKE, AKS, K3s, minikube, etc.)

**Verificar:**

```bash
kubectl version --short
kubectl cluster-info
```

### 2. Ferramentas de Build

- **Go**: 1.21 ou superior (para build local)
- **Docker ou Podman**: Para construir a imagem
- **Make**: Para automação de tarefas

**Verificar:**

```bash
go version
docker --version
make --version
```

### 3. Conta AWS

- **Conta AWS** com permissões de administrador (para configuração inicial)
- **AWS CLI** configurada (opcional mas recomendado)

**Verificar:**

```bash
aws sts get-caller-identity
```

---

## Construindo o Operator

### Opção 1: Build Local + Docker

**Comando:**

```bash
# 1. Navegar para o diretório do projeto
cd /Users/andrebassi/works/.solutions/operators/infra-operator

# 2. Baixar dependências Go
make mod-download

# 3. Construir binário (opcional - para teste local)
make build
./bin/manager --help

# 4. Construir imagem Docker
make docker-build IMG=infra-operator:v1.0.0

# 5. Tagear para registry (exemplo com ttl.sh - registry efêmero)
docker tag infra-operator:v1.0.0 ttl.sh/infra-operator:v1.0.0

# 6. Push para registry
docker push ttl.sh/infra-operator:v1.0.0
```

### Opção 2: Build Multi-arch com Buildx

**Comando:**

```bash
# Construir e fazer push para múltiplas arquiteturas
make docker-buildx REGISTRY=ttl.sh IMG=infra-operator:v1.0.0
```

### Opção 3: Build para Registry Privado

**Comando:**

```bash
# Login no registry
docker login registry.example.com

# Build e push
docker build -t registry.example.com/infra-operator:v1.0.0 .
docker push registry.example.com/infra-operator:v1.0.0

# Atualizar deployment
# Editar config/manager/deployment.yaml:
#   image: registry.example.com/infra-operator:v1.0.0
```

---

## Configuração IAM/IRSA

### Para EKS com IRSA (Recomendado)

#### 1. Criar Política IAM

**Comando:**

```bash
# Criar arquivo policy.json
cat > /tmp/infra-operator-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Sid": "STSPermissions",
      "Effect": "Allow",
      "Action": ["sts:GetCallerIdentity"],
      "Resource": "*"
},
{
      "Sid": "S3FullAccess",
      "Effect": "Allow",
      "Action": ["s3:*"],
      "Resource": "*"
},
{
      "Sid": "RDSFullAccess",
      "Effect": "Allow",
      "Action": ["rds:*"],
      "Resource": "*"
},
{
      "Sid": "EC2FullAccess",
      "Effect": "Allow",
      "Action": ["ec2:*"],
      "Resource": "*"
},
{
      "Sid": "SQSFullAccess",
      "Effect": "Allow",
      "Action": ["sqs:*"],
      "Resource": "*"
}
  ]
}
EOF

# Criar a política
aws iam create-policy \
  --policy-name InfraOperatorPolicy \
  --policy-document file:///tmp/infra-operator-policy.json
```

#### 2. Criar Role IAM com Trust Policy para IRSA

**Comando:**

```bash
# Obter provedor OIDC do cluster EKS
CLUSTER_NAME=your-cluster-name
OIDC_PROVIDER=$(aws eks describe-cluster \
  --name $CLUSTER_NAME \
  --query "cluster.identity.oidc.issuer" \
  --output text | sed -e 's|^https://||')

# Criar trust policy
cat > /tmp/trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
{
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):oidc-provider/${OIDC_PROVIDER}"
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

# Criar role
aws iam create-role \
  --role-name infra-operator-role \
  --assume-role-policy-document file:///tmp/trust-policy.json

# Anexar política ao role
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
aws iam attach-role-policy \
  --role-name infra-operator-role \
  --policy-arn arn:aws:iam::${ACCOUNT_ID}:policy/InfraOperatorPolicy
```

#### 3. Anotar Service Account

**Comando:**

```bash
# Editar config/rbac/service_account.yaml
# Descomentar e ajustar a anotação:

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

cat > config/rbac/service_account.yaml <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: infra-operator-controller-manager
  namespace: infra-operator-system
  annotations:
eks.amazonaws.com/role-arn: arn:aws:iam::${ACCOUNT_ID}:role/infra-operator-role
EOF
```

### Para Kubernetes Não-EKS (Credenciais Estáticas)

**Comando:**

```bash
# Criar Secret com credenciais AWS
kubectl create namespace infra-operator-system

kubectl create secret generic aws-credentials \
  -n infra-operator-system \
  --from-literal=access-key-id=AKIAXXXXXXXXXXXXXXXX \
  --from-literal=secret-access-key=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

---

## Instalação no Cluster

### Instalação Completa (Método Automatizado)

**Comando:**

```bash
# Instala CRDs e faz deploy do operator
make install-complete
```

### Instalação Passo a Passo (Método Manual)

#### 1. Criar Namespace

**Comando:**

```bash
kubectl apply -f config/manager/namespace.yaml
```

#### 2. Instalar CRDs

**Comando:**

```bash
kubectl apply -f config/crd/bases/aws-infra-operator.runner.codes_awsproviders.yaml
kubectl apply -f config/crd/bases/aws-infra-operator.runner.codes_s3buckets.yaml

# Verificar instalação
kubectl get crds | grep aws-infra-operator.runner.codes
```

#### 3. Configurar RBAC

**Comando:**

```bash
kubectl apply -f config/rbac/service_account.yaml
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
```

#### 4. Deploy do Operator

**Comando:**

```bash
# Se você fez push para um registry diferente, edite primeiro:
# vim config/manager/deployment.yaml
# Alterar a linha: image: infra-operator:latest
# Para: image: ttl.sh/infra-operator:v1.0.0

kubectl apply -f config/manager/deployment.yaml
```

---

## Verificação

### 1. Verificar Pods

**Comando:**

```bash
# Aguardar pod ficar Running
kubectl get pods -n infra-operator-system

# Exemplo de output esperado:
# NAME                                              READY   STATUS    RESTARTS   AGE
# infra-operator-controller-manager-xxxxxxxxxx-xxxxx   1/1     Running   0          30s
```

### 2. Verificar Logs

**Comando:**

```bash
# Visualizar logs do operator
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager \
  -f

# Procurar por:
# - "starting manager"
# - Sem erros de autenticação
```

### 3. Verificar CRDs

**Comando:**

```bash
# Listar CRDs instalados
kubectl get crds | grep aws-infra-operator.runner.codes

# Testar explain (verifica schema)
kubectl explain awsprovider.spec
kubectl explain s3bucket.spec
```

### 4. Health Checks

**Comando:**

```bash
# Port-forward para endpoint de saúde
kubectl port-forward -n infra-operator-system \
  deploy/infra-operator-controller-manager 8081:8081

# Em outro terminal, testar endpoints
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz

# Ambos devem retornar: ok
```

---

## Primeiro Recurso

### 1. Criar AWSProvider

**Comando:**

```bash
# Para IRSA (EKS)
cat <<EOF | kubectl apply -f -
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-default
  namespace: default
spec:
  region: us-east-1
  roleARN: arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):role/infra-operator-role
  defaultTags:
    managed-by: infra-operator
    environment: test
EOF
```

OU para credenciais estáticas:

**Exemplo:**

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: default
type: Opaque
stringData:
  access-key-id: AKIAXXXXXXXXXXXXXXXX
  secret-access-key: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
---
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-default
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
EOF
```

### 2. Verificar Provider

**Comando:**

```bash
# Aguardar provider ficar Ready
kubectl get awsprovider aws-default -w

# Visualizar detalhes
kubectl describe awsprovider aws-default

# Verificar status.ready = true e accountID está populado
kubectl get awsprovider aws-default -o jsonpath='{.status.ready}'
kubectl get awsprovider aws-default -o jsonpath='{.status.accountID}'
```

### 3. Criar Bucket S3

**Comando:**

```bash
cat <<EOF | kubectl apply -f -
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: test-bucket
  namespace: default
spec:
  providerRef:
    name: aws-default
  bucketName: infra-operator-test-$(date +%s)
  encryption:
    algorithm: AES256
  publicAccessBlock:
    blockPublicAcls: true
    ignorePublicAcls: true
    blockPublicPolicy: true
    restrictPublicBuckets: true
  deletionPolicy: Delete
EOF
```

### 4. Verificar Bucket

**Comando:**

```bash
# Observar status do bucket
kubectl get s3bucket test-bucket -w

# Visualizar detalhes
kubectl describe s3bucket test-bucket

# Verificar na AWS
BUCKET_NAME=$(kubectl get s3bucket test-bucket -o jsonpath='{.spec.bucketName}')
aws s3 ls | grep $BUCKET_NAME
```

### 5. Testar Deleção

**Comando:**

```bash
# Deletar bucket
kubectl delete s3bucket test-bucket

# Verificar se foi removido da AWS
aws s3 ls | grep $BUCKET_NAME || echo "Bucket deleted successfully"
```

---

## Troubleshooting

### Problema: Pod não inicia

**Comando:**

```bash
# Verificar eventos
kubectl describe pod -n infra-operator-system \
  -l control-plane=controller-manager

# Verificar logs
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager --previous

# Causas comuns:
# - Imagem não encontrada (ImagePullBackOff)
# - Recursos insuficientes
# - RBAC incorreto
```

### Problema: AWSProvider não Ready

**Comando:**

```bash
# Visualizar logs do operator
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager | grep -i error

# Visualizar status do provider
kubectl describe awsprovider <name>

# Causas comuns:
# - IRSA mal configurado (trust policy incorreto)
# - Service account sem anotação
# - Credenciais inválidas
# - Região inválida
```

### Problema: S3Bucket não cria

**Comando:**

```bash
# Visualizar eventos do bucket
kubectl describe s3bucket <name>

# Visualizar logs do operator
kubectl logs -n infra-operator-system \
  -l control-plane=controller-manager | grep -i s3

# Causas comuns:
# - Nome do bucket já existe (deve ser globalmente único)
# - AWSProvider não está Ready
# - Permissões IAM insuficientes
# - Região incorreta
```

### Problema: Recurso não deleta

**Comando:**

```bash
# Visualizar finalizers
kubectl get s3bucket <name> -o yaml | grep finalizers -A 5

# Forçar remoção de finalizer
kubectl patch s3bucket <name> \
  -p '{"metadata":{"finalizers":[]}}' \
  --type=merge

# Deletar novamente
kubectl delete s3bucket <name>
```

### Debug com Port Forward

**Comando:**

```bash
# Expor endpoint de métricas
kubectl port-forward -n infra-operator-system \
  deploy/infra-operator-controller-manager 8080:8080

# Visualizar métricas (se configurado)
curl http://localhost:8080/metrics
```

---

## Desinstalação

### Remover Recursos Criados

**Comando:**

```bash
# Deletar todos os buckets
kubectl delete s3buckets --all -A

# Deletar todos os providers
kubectl delete awsproviders --all -A
```

### Remover Operator

**Comando:**

```bash
# Método automatizado
make uninstall-complete

# Ou manual:
kubectl delete -f config/manager/deployment.yaml
kubectl delete -f config/rbac/
kubectl delete -f config/crd/bases/
kubectl delete namespace infra-operator-system
```

---

## Próximos Passos

1. **Produção**: Revisar permissões IAM (princípio do menor privilégio)
2. **GitOps**: Integrar com ArgoCD ou Flux
3. **Monitoramento**: Configurar métricas Prometheus
4. **Alertas**: Configurar alertas para recursos em estado NotReady
5. **Backup**: Implementar backup de CRs importantes

Para mais informações, veja:
- [Introdução](/) - Visão geral do projeto
- [Quickstart](/quickstart) - Guia de início rápido
- [Serviços AWS](/services/networking/vpc) - Documentação dos serviços
