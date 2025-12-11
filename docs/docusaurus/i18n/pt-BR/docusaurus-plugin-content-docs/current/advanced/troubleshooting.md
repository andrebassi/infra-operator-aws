---
title: 'Troubleshooting'
description: 'Problemas comuns e soluções para o Infra Operator'
sidebar_position: 3
---

# Troubleshooting

Este guia cobre problemas comuns que você pode encontrar ao usar o Infra Operator e como resolvê-los.

## Comandos de Diagnóstico

### Verificar Status do Operator

**Comando:**

```bash
# Verificar se operator está rodando
kubectl get pods -n infra-operator

# Verificar logs do operator
kubectl logs -n infra-operator deploy/infra-operator --tail=100

# Acompanhar logs em tempo real
kubectl logs -n infra-operator deploy/infra-operator -f
```

### Verificar Status dos Recursos

**Comando:**

```bash
# Listar todos os recursos
kubectl get vpc,subnet,sg,s3,ec2 -A

# Obter status detalhado
kubectl describe vpc my-vpc

# Verificar eventos
kubectl get events -n infra-operator --sort-by='.lastTimestamp'
```

### Verificar AWSProvider

**Comando:**

```bash
# Verificar se provider está ready
kubectl get awsprovider

# Verificar detalhes do provider
kubectl describe awsprovider aws-production
```

## Problemas Comuns

### Operator Não Está Iniciando

**Sintomas:** Pod do operator em estado CrashLoopBackOff ou Error

**Verificar logs:**

```bash
kubectl logs -n infra-operator deploy/infra-operator --previous
```

**Causas comuns:**

1. **CRDs Faltando**

   **Comando:**

   ```bash
   # Verificar se CRDs estão instalados
   kubectl get crds | grep aws-infra-operator.runner.codes

   # Reinstalar se estiver faltando
   kubectl apply -f chart/crds/
   ```

2. **RBAC Inválido**

   **Comando:**

   ```bash
   # Verificar ServiceAccount
   kubectl get sa -n infra-operator

   # Verificar ClusterRole
   kubectl get clusterrole | grep infra-operator
   ```

3. **Limites de recursos muito baixos**

   **Exemplo:**

   ```yaml
   # Aumentar em values.yaml
   operator:
     resources:
       limits:
         memory: 512Mi
       requests:
         memory: 256Mi
   ```

### AWSProvider Não Ready

**Sintomas:** AWSProvider com `ready: false`

**Verificar:**

```bash
kubectl describe awsprovider aws-production
```

**Causas comuns:**

1. **Credenciais inválidas**

   **Comando:**

   ```bash
   # Verificar se Secret existe
   kubectl get secret aws-credentials -n infra-operator

   # Testar credenciais
   AWS_ACCESS_KEY_ID=$(kubectl get secret aws-credentials -n infra-operator \
     -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 -d)
   AWS_SECRET_ACCESS_KEY=$(kubectl get secret aws-credentials -n infra-operator \
     -o jsonpath='{.data.AWS_SECRET_ACCESS_KEY}' | base64 -d)
   aws sts get-caller-identity
   ```

2. **Região errada**

   **Exemplo:**

   ```yaml
   # Verificar região no provider
   spec:
     region: us-east-1  # Deve corresponder aos seus recursos AWS
   ```

3. **IRSA não configurado (EKS)**

   **Comando:**

   ```bash
   # Verificar anotação do ServiceAccount
   kubectl get sa infra-operator -n infra-operator -o yaml | grep eks.amazonaws.com

   # Verificar trust policy do IAM role
   aws iam get-role --role-name infra-operator-role
   ```

### Recurso Preso em Pending

**Sintomas:** Recurso permanece em estado `pending` indefinidamente

**Verificar:**

```bash
kubectl describe vpc my-vpc
kubectl logs -n infra-operator deploy/infra-operator | grep "my-vpc"
```

**Causas comuns:**

1. **AWSProvider não ready**

   **Comando:**

   ```bash
   kubectl get awsprovider
   # Garantir que o provider referenciado pelo recurso está ready
   ```

2. **Erros de API AWS**

   **Comando:**

   ```bash
   # Verificar logs do operator por erros AWS
   kubectl logs -n infra-operator deploy/infra-operator | grep -i "error\|failed"
   ```

3. **Rate limiting**

   **Comando:**

   ```bash
   # Procurar por erros de throttling
   kubectl logs -n infra-operator deploy/infra-operator | grep -i "throttl"
   ```

### Recurso Não Deleta

**Sintomas:** Recurso preso em estado `Terminating`

**Verificar:**

```bash
kubectl get vpc my-vpc -o yaml | grep -A 10 finalizers
```

**Soluções:**

1. **Verificar recursos dependentes**

   **Comando:**

   ```bash
   # VPC não pode ser deletado se tiver subnets, IGWs, etc.
   aws ec2 describe-subnets --filters "Name=vpc-id,Values=vpc-xxx"
   aws ec2 describe-internet-gateways --filters "Name=attachment.vpc-id,Values=vpc-xxx"
   ```

2. **Remover finalizer (último recurso)**

   **Comando:**

   ```bash
   # AVISO: Isso pode deixar recursos AWS órfãos
   kubectl patch vpc my-vpc -p '{"metadata":{"finalizers":[]}}' --type=merge
   ```

3. **Forçar deleção com timeout**

   **Comando:**

   ```bash
   kubectl delete vpc my-vpc --timeout=30s
   ```

### Drift Detectado

**Sintomas:** Recurso mostra drift entre spec do Kubernetes e AWS

**Verificar:**

```bash
kubectl describe vpc my-vpc | grep -A 5 "Drift"
```

**Soluções:**

1. **Atualizar spec para corresponder à AWS**

   **Comando:**

   ```bash
   # Obter estado atual da AWS
   aws ec2 describe-vpcs --vpc-ids vpc-xxx

   # Atualizar recurso Kubernetes para corresponder
   kubectl edit vpc my-vpc
   ```

2. **Forçar reconciliação**

   **Comando:**

   ```bash
   # Adicionar anotação para disparar reconcile
   kubectl annotate vpc my-vpc force-reconcile="$(date +%s)" --overwrite
   ```

3. **Habilitar auto-remediação**

   **Exemplo:**

   ```yaml
   spec:
     driftDetection:
       enabled: true
       autoRemediate: true
   ```

### EC2 Instance Não Inicia

**Sintomas:** EC2Instance presa em `pending` ou `stopped`

**Verificar:**

```bash
kubectl describe ec2instance my-instance
aws ec2 describe-instances --instance-ids i-xxx
```

**Causas comuns:**

1. **AMI inválida**

   **Comando:**

   ```bash
   # Verificar se AMI existe na região
   aws ec2 describe-images --image-ids ami-xxx
   ```

2. **Tipo de instância inválido**

   **Comando:**

   ```bash
   # Verificar tipos de instância disponíveis
   aws ec2 describe-instance-types --instance-types t3.micro
   ```

3. **Problemas com Subnet/Security Group**

   **Comando:**

   ```bash
   # Verificar se subnet existe
   aws ec2 describe-subnets --subnet-ids subnet-xxx

   # Verificar security group
   aws ec2 describe-security-groups --group-ids sg-xxx
   ```

4. **Capacidade insuficiente**

   **Comando:**

   ```bash
   # Tentar AZ diferente ou tipo de instância
   aws ec2 describe-instance-type-offerings \
     --location-type availability-zone \
     --filters Name=instance-type,Values=t3.micro
   ```

### S3 Bucket Permissão Negada

**Sintomas:** Criação de S3Bucket falha com access denied

**Verificar:**

```bash
kubectl logs -n infra-operator deploy/infra-operator | grep "s3\|bucket"
```

**Soluções:**

1. **Verificar permissões IAM**

   **JSON:**

   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "s3:CreateBucket",
           "s3:DeleteBucket",
           "s3:GetBucketLocation",
           "s3:GetBucketTagging",
           "s3:PutBucketTagging",
           "s3:GetBucketVersioning",
           "s3:PutBucketVersioning",
           "s3:GetEncryptionConfiguration",
           "s3:PutEncryptionConfiguration"
         ],
         "Resource": "*"
       }
     ]
   }
   ```

2. **Nome do bucket já existe**

   **Comando:**

   ```bash
   # Nomes de bucket S3 são globalmente únicos
   aws s3api head-bucket --bucket my-bucket-name
   ```

### Problemas de Conexão com LocalStack

**Sintomas:** Recursos falham ao usar LocalStack

**Verificar:**

```bash
# Verificar se LocalStack está rodando
kubectl get pods | grep localstack

# Testar conectividade
kubectl run test --rm -it --image=curlimages/curl -- \
  curl http://localstack.default.svc.cluster.local:4566/_localstack/health
```

**Soluções:**

1. **Verificar URL do endpoint**

   **Exemplo:**

   ```yaml
   # No AWSProvider
   spec:
     endpoint: http://localstack.default.svc.cluster.local:4566
   ```

2. **Verificar serviços do LocalStack**

   **Comando:**

   ```bash
   # Listar serviços em execução
   curl http://localhost:4566/_localstack/health | jq
   ```

## Problemas de Performance

### Reconciliação Lenta

**Sintomas:** Recursos levam muito tempo para sincronizar

**Soluções:**

1. **Aumentar concorrência**

   **Exemplo:**

   ```yaml
   # No deployment do operator
   args:
     - --max-concurrent-reconciles=10
   ```

2. **Verificar rate limits**

   **Comando:**

   ```bash
   # Monitorar chamadas de API AWS
   kubectl logs -n infra-operator deploy/infra-operator | grep -i "rate\|limit"
   ```

### Alto Uso de Memória

**Sintomas:** Operator usando memória excessiva

**Soluções:**

1. **Aumentar limites de memória**

   **Exemplo:**

   ```yaml
   operator:
     resources:
       limits:
         memory: 1Gi
   ```

2. **Reduzir tamanho do cache**

   **Exemplo:**

   ```yaml
   args:
     - --cache-size=100
   ```

## Obtendo Ajuda

### Coletar Informações de Debug

**Comando:**

```bash
# Criar bundle de debug
mkdir debug-bundle
kubectl get pods -n infra-operator -o yaml > debug-bundle/pods.yaml
kubectl logs -n infra-operator deploy/infra-operator > debug-bundle/logs.txt
kubectl get crds | grep aws-infra-operator.runner.codes > debug-bundle/crds.txt
kubectl get awsprovider,vpc,subnet,sg -A -o yaml > debug-bundle/resources.yaml
kubectl get events -n infra-operator > debug-bundle/events.txt
```

### Reportar Problemas

Ao reportar problemas, inclua:

1. Versão do operator
2. Versão do Kubernetes
3. Provedor de cloud (AWS/LocalStack)
4. YAML do recurso (credenciais removidas)
5. Logs do operator
6. Mensagens de erro

**GitHub Issues:** https://github.com/andrebassi/infra-operator/issues
