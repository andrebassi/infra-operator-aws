---
title: 'Especificações dos CRDs'
description: 'Especificações completas dos CRDs para todos os recursos do Infra Operator'
sidebar_position: 3
---

# Especificações dos CRDs

Esta página fornece especificações detalhadas para todos os Custom Resource Definitions (CRDs) no Infra Operator.

## Instalando CRDs

Os CRDs são automaticamente instalados quando você faz deploy do Infra Operator via Helm:

**Comando:**

```bash
helm install infra-operator ./chart -n infra-operator --create-namespace
```

Para verificar se os CRDs estão instalados:

**Comando:**

```bash
kubectl get crds | grep aws-infra-operator.runner.codes
```

**Output esperado (30 CRDs):**

```
albs.aws-infra-operator.runner.codes
apigateways.aws-infra-operator.runner.codes
awsproviders.aws-infra-operator.runner.codes
certificates.aws-infra-operator.runner.codes
cloudfronts.aws-infra-operator.runner.codes
computestacks.aws-infra-operator.runner.codes
dynamodbtables.aws-infra-operator.runner.codes
ec2instances.aws-infra-operator.runner.codes
ec2keypairs.aws-infra-operator.runner.codes
ecrrepositories.aws-infra-operator.runner.codes
ecsclusters.aws-infra-operator.runner.codes
eksclusters.aws-infra-operator.runner.codes
elasticacheclusters.aws-infra-operator.runner.codes
elasticips.aws-infra-operator.runner.codes
iamroles.aws-infra-operator.runner.codes
internetgateways.aws-infra-operator.runner.codes
kmskeys.aws-infra-operator.runner.codes
lambdafunctions.aws-infra-operator.runner.codes
natgateways.aws-infra-operator.runner.codes
nlbs.aws-infra-operator.runner.codes
rdsinstances.aws-infra-operator.runner.codes
route53hostedzones.aws-infra-operator.runner.codes
route53recordsets.aws-infra-operator.runner.codes
routetables.aws-infra-operator.runner.codes
s3buckets.aws-infra-operator.runner.codes
secretsmanagersecrets.aws-infra-operator.runner.codes
securitygroups.aws-infra-operator.runner.codes
snstopics.aws-infra-operator.runner.codes
sqsqueues.aws-infra-operator.runner.codes
subnets.aws-infra-operator.runner.codes
vpcs.aws-infra-operator.runner.codes
```

## Convenção de Nomenclatura dos CRDs

| Kubernetes Kind | Plural | Nome Curto |
|----------------|--------|------------|
| VPC | vpcs | vpc |
| Subnet | subnets | subnet |
| InternetGateway | internetgateways | igw |
| NATGateway | natgateways | nat |
| RouteTable | routetables | rt |
| SecurityGroup | securitygroups | sg |
| ElasticIP | elasticips | eip |
| ALB | albs | alb |
| NLB | nlbs | nlb |
| EC2Instance | ec2instances | ec2 |
| EKSCluster | eksclusters | eks |
| ECSCluster | ecsclusters | ecs |
| LambdaFunction | lambdafunctions | lambda |
| S3Bucket | s3buckets | s3 |
| RDSInstance | rdsinstances | rds |
| DynamoDBTable | dynamodbtables | ddb |
| ElastiCacheCluster | elasticacheclusters | ec |
| ECRRepository | ecrrepositories | ecr |
| SQSQueue | sqsqueues | sqs |
| SNSTopic | snstopics | sns |
| IAMRole | iamroles | iam |
| KMSKey | kmskeys | kms |
| SecretsManagerSecret | secretsmanagersecrets | secret |
| Certificate | certificates | cert |
| CloudFront | cloudfronts | cf |
| APIGateway | apigateways | apigw |
| Route53HostedZone | route53hostedzones | r53hz |
| Route53RecordSet | route53recordsets | r53rs |
| ComputeStack | computestacks | cs |
| AWSProvider | awsproviders | provider |

## Usando Nomes Curtos

**Comando:**

```bash
# Estes são equivalentes:
kubectl get vpcs
kubectl get vpc

kubectl get securitygroups
kubectl get sg

kubectl get s3buckets
kubectl get s3
```

## Visualizando Schema dos CRDs

**Comando:**

```bash
# Ver definição completa do CRD
kubectl get crd vpcs.aws-infra-operator.runner.codes -o yaml

# Explicar campos da spec
kubectl explain vpc.spec

# Explicar campos aninhados
kubectl explain vpc.spec.tags
```

## Printer Columns

Cada CRD define colunas customizadas para `kubectl get`:

### VPC

**Comando:**

```bash
kubectl get vpc
```

```
NAME     VPC-ID                 CIDR          STATE      READY   AGE
my-vpc   vpc-0123456789abcdef0  10.0.0.0/16   available  true    5m
```

### S3Bucket

**Comando:**

```bash
kubectl get s3
```

```
NAME        BUCKET-NAME              REGION      VERSIONING   READY   AGE
my-bucket   my-bucket-production     us-east-1   Enabled      true    10m
```

### EC2Instance

**Comando:**

```bash
kubectl get ec2
```

```
NAME         INSTANCE-ID          TYPE       STATE     PUBLIC-IP      READY   AGE
my-instance  i-0123456789abcdef0  t3.micro   running   54.123.45.67   true    15m
```

## Validação

Todos os CRDs incluem regras de validação aplicadas pelo API server do Kubernetes:

### Campos Obrigatórios

**Exemplo:**

```yaml
# Isto vai falhar - falta cidrBlock
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: invalid-vpc
spec:
  providerRef:
    name: aws-production
  # cidrBlock é obrigatório!
```

### Validação de Padrão

**Exemplo:**

```yaml
# Isto vai falhar - formato de CIDR inválido
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: invalid-vpc
spec:
  providerRef:
    name: aws-production
  cidrBlock: "invalid"  # Deve ser notação CIDR válida
```

### Validação de Enum

**Exemplo:**

```yaml
# Isto vai falhar - deletionPolicy inválida
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
metadata:
  name: invalid-vpc
spec:
  providerRef:
    name: aws-production
  cidrBlock: "10.0.0.0/16"
  deletionPolicy: Invalid  # Deve ser Delete, Retain ou Orphan
```

## Webhooks (Opcional)

O Infra Operator inclui webhooks de validação para validação adicional:

- Validação entre campos
- Restrições específicas da AWS
- Verificações de dependência de recursos

Habilitar webhooks nos values do Helm:

**Exemplo:**

```yaml
webhooks:
  enabled: true
```

:::note
Webhooks requerem cert-manager para certificados TLS.
:::

## Atualizando CRDs

Ao atualizar o Infra Operator, os CRDs são automaticamente atualizados:

**Comando:**

```bash
# Atualizar release Helm
helm upgrade infra-operator ./chart -n infra-operator

# Verificar se CRDs foram atualizados
kubectl get crd vpcs.aws-infra-operator.runner.codes -o jsonpath='{.metadata.annotations.controller-gen\.kubebuilder\.io/version}'
```

## Backup e Restauração

### Fazer Backup de CRDs e Recursos

**Comando:**

```bash
# Backup dos CRDs
kubectl get crds -o yaml | grep -A 1000 "aws-infra-operator.runner.codes" > crds-backup.yaml

# Backup de todos os recursos
for resource in vpc subnet sg s3 ec2 rds; do
  kubectl get $resource -A -o yaml > ${resource}-backup.yaml
done
```

### Restaurar

**Comando:**

```bash
# Restaurar CRDs (se necessário)
kubectl apply -f crds-backup.yaml

# Restaurar recursos
kubectl apply -f vpc-backup.yaml
kubectl apply -f subnet-backup.yaml
# ... etc
```
