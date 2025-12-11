---
title: 'Instalação'
description: 'Instale o Infra Operator via Helm Chart'
---

## Pré-requisitos

Antes de instalar o Infra Operator, certifique-se de ter:

- **Kubernetes 1.28+** - Cluster Kubernetes funcionando
- **Helm 3.x** - Gerenciador de pacotes Kubernetes
- **kubectl** - CLI Kubernetes configurado
- **Credenciais AWS** - Credenciais AWS ou LocalStack para testes

:::tip

Para desenvolvimento local, recomendamos usar **LocalStack** para emular serviços AWS sem custo.

:::

## Instalação via Helm

### 1. Clone o Repositório

**Comando:**

```bash
git clone https://github.com/andrebassi/infra-operator-aws.git
cd infra-operator
```

### 2. Instale via Helm

**Comando:**

```bash
# Instalar com valores padrão
helm install infra-operator ./chart \
  --namespace iop-system \
  --create-namespace

# Ou com valores personalizados
helm install infra-operator ./chart \
  --namespace iop-system \
  --create-namespace \
  --set image.tag=latest \
  --set replicaCount=2 \
  --set resources.requests.memory=256Mi
```

### 3. Verificar Instalação

**Comando:**

```bash
# Verificar pods
kubectl get pods -n iop-system

# Deve mostrar algo como:
# NAME                              READY   STATUS    RESTARTS   AGE
# infra-operator-5d4c7b9f8d-xxxxx   1/1     Running   0          30s

# Verificar CRDs instalados (26 CRDs)
kubectl get crds | grep aws-infra-operator.runner.codes

# Verificar logs
kubectl logs -n iop-system deploy/infra-operator
```

<Check>
  Se todos os pods estão **Running** e todos os 26 CRDs foram criados, a instalação foi bem-sucedida!
</Check>

## Configuração do AWSProvider

Antes de criar recursos AWS, você precisa configurar as credenciais:

### Opção 1: IRSA (Recomendado para EKS)

**Exemplo:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-provider
spec:
  region: us-east-1
  roleARN: arn:aws:iam::123456789012:role/infra-operator-role
```

### Opção 2: Credenciais Estáticas

**Comando:**

```bash
# Criar secret com credenciais AWS
kubectl create secret generic aws-credentials \
  --from-literal=aws-access-key-id=YOUR_ACCESS_KEY \
  --from-literal=aws-secret-access-key=YOUR_SECRET_KEY \
  -n iop-system
```

**Exemplo:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: aws-provider
spec:
  region: us-east-1
  credentials:
    secretRef:
      name: aws-credentials
```

### Opção 3: LocalStack (Desenvolvimento)

**Comando:**

```bash
# Instalar LocalStack
docker run -d \
  --name localstack \
  -p 4566:4566 \
  -e SERVICES=ec2,s3,dynamodb,sqs,sns,iam,secretsmanager,kms \
  localstack/localstack:latest
```

**Exemplo:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: AWSProvider
metadata:
  name: localstack-provider
spec:
  region: us-east-1
  endpoint: http://localstack:4566
  credentials:
    secretRef:
      name: localstack-credentials
```

:::note

Para LocalStack, use credenciais falsas: `test` / `test`

:::

## Aplicar o Provider

**Comando:**

```bash
kubectl apply -f awsprovider.yaml

# Verificar status
kubectl get awsprovider aws-provider

# Deve mostrar:
# NAME           REGION      READY   AGE
# aws-provider   us-east-1   True    10s
```

## Testando com Samples

O repositório inclui **26 samples** prontos na pasta `./samples/`, organizados na ordem de criação recomendada:

### Estrutura dos Samples

#### Rede (01-09)

**01-vpc.yaml** - Virtual Private Cloud:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: VPC
spec:
  cidrBlock: "10.10.0.0/16"          # Faixa de IP privado
  enableDnsSupport: true              # Habilitar resolução DNS
  enableDnsHostnames: true            # Habilitar hostnames DNS
```
Cria a base de rede AWS. Todos os outros recursos de rede dependem da VPC.

**02-subnet.yaml** - Subnet dentro da VPC:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Subnet
spec:
  vpcID: "vpc-xxx"                    # Referência à VPC criada
  cidrBlock: "10.10.1.0/24"          # Sub-faixa da VPC
  availabilityZone: us-east-1a        # AZ específica
  mapPublicIpOnLaunch: true          # Atribuir IP público automaticamente
```
Cria uma subnet pública para hospedar recursos acessíveis pela internet.

**03-elastic-ip.yaml** - IP Público Estático:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElasticIP
spec:
  domain: vpc                         # IP para uso em VPC
```
IP público fixo, usado para NAT Gateways ou instâncias EC2.

**04-internet-gateway.yaml** - Internet Gateway:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: InternetGateway
spec:
  vpcID: "vpc-xxx"                    # Anexado à VPC
```
Permite que recursos da VPC acessem a internet e sejam acessados externamente.

**05-nat-gateway.yaml** - NAT Gateway:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NATGateway
spec:
  subnetID: "subnet-xxx"              # Subnet pública
  allocationID: ""                    # Alocação do Elastic IP
```
Permite que instâncias em subnets privadas acessem a internet (apenas saída).

**06-route-table.yaml** - Tabela de Rotas:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RouteTable
spec:
  vpcID: "vpc-xxx"
  routes:
    - destinationCIDR: 0.0.0.0/0      # Rota padrão
      gatewayID: ""                    # ID do Internet Gateway
```
Define rotas de tráfego de rede (ex: para internet via IGW).

**07-security-group.yaml** - Firewall Virtual:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecurityGroup
spec:
  vpcID: "vpc-xxx"
  ingressRules:                       # Tráfego de entrada
    - ipProtocol: tcp
      fromPort: 80
      toPort: 80
      cidrIPv4: 0.0.0.0/0            # Permitir HTTP de qualquer lugar
  egressRules:                        # Tráfego de saída
    - ipProtocol: "-1"                # Todos os protocolos
      cidrIPv4: 0.0.0.0/0            # Para qualquer destino
```
Controla o tráfego de rede permitido para recursos AWS.

**08-alb.yaml** - Application Load Balancer (Camada 7):
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ALB
spec:
  loadBalancerName: helm-test-alb
  scheme: internet-facing             # Público ou interno
  subnets: ["subnet-xxx"]             # Múltiplas subnets (HA)
  ipAddressType: ipv4
```
Load balancer para HTTP/HTTPS com roteamento baseado em conteúdo.

**09-nlb.yaml** - Network Load Balancer (Camada 4):
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: NLB
spec:
  loadBalancerName: helm-test-nlb
  scheme: internet-facing
  ipAddressType: ipv4
```
Load balancer para TCP/UDP com alta performance e baixa latência.

#### Segurança (10-13)

**10-iam-role.yaml** - IAM Role:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: IAMRole
spec:
  roleName: helm-test-role
  assumeRolePolicyDocument: |         # Política de confiança
{
      "Statement": [{
        "Effect": "Allow",
        "Principal": {"Service": "lambda.amazonaws.com"},
        "Action": "sts:AssumeRole"
      }]
}
```
Role IAM para funções Lambda assumirem e acessarem recursos AWS.

**11-kms-key.yaml** - Chave de Criptografia KMS:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: KMSKey
spec:
  description: "Chave KMS de teste Helm"
  keyUsage: ENCRYPT_DECRYPT           # Criptografia/Descriptografia
```
Chave de criptografia gerenciada para proteger dados sensíveis.

**12-secrets-manager.yaml** - Secrets Manager:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SecretsManagerSecret
spec:
  secretName: helm-test-secret
  secretString: '{"username":"admin","password":"secret123"}'
```
Armazena credenciais, chaves de API e outros segredos com rotação automática.

**13-certificate.yaml** - Certificado SSL/TLS ACM:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: Certificate
spec:
  domainName: example.com
  subjectAlternativeNames:            # SANs
- "*.example.com"                 # Wildcard
  validationMethod: DNS               # DNS ou EMAIL
```
Certificado SSL/TLS para HTTPS em ALB, CloudFront, API Gateway.

#### Armazenamento & Banco de Dados (14-18)

**14-s3.yaml** - Bucket S3:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
spec:
  bucketName: helm-test-bucket
  versioning:
    enabled: true                     # Versionamento de objetos
  encryption:
    algorithm: AES256                 # Criptografia em repouso
```
Armazenamento de objetos para arquivos, backups, data lakes, hospedagem estática.

**15-ecr-repository.yaml** - Registro de Containers:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECRRepository
spec:
  repositoryName: helm-test-repo
  imageTagMutability: MUTABLE
  imageScanningConfiguration:
    scanOnPush: true                  # Scan de vulnerabilidades
```
Registro privado para imagens Docker/OCI.

**16-dynamodb.yaml** - Tabela DynamoDB NoSQL:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: DynamoDBTable
spec:
  tableName: helm-test-users
  hashKey:
    name: userID                      # Chave de partição
    type: S                           # Tipo String
  billingMode: PAY_PER_REQUEST        # Preço sob demanda
```
Banco de dados NoSQL serverless com alta performance e escalabilidade automática.

**17-rds-instance.yaml** - Banco de Dados Relacional RDS:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
spec:
  dbInstanceIdentifier: helm-test-db
  dbInstanceClass: db.t3.micro        # Tipo da instância
  engine: postgres                    # postgres, mysql, mariadb, etc
  engineVersion: "14.5"
  masterUsername: admin
  masterUserPassword: secret123
  allocatedStorage: 20                # GB
  storageType: gp2                    # SSD
```
Banco de dados relacional gerenciado (PostgreSQL, MySQL, etc) com backups automáticos.

**18-elasticache.yaml** - ElastiCache Redis/Memcached:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ElastiCacheCluster
spec:
  cacheClusterID: helm-test-cache
  engine: redis                       # redis ou memcached
  cacheNodeType: cache.t3.micro
  numCacheNodes: 1
```
Cache em memória gerenciado para melhorar performance de aplicações.

#### Mensageria (19-20)

**19-sqs.yaml** - Fila SQS:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SQSQueue
spec:
  queueName: helm-test-queue
  messageRetentionPeriod: 345600      # 4 dias (segundos)
  visibilityTimeout: 30               # 30 segundos
```
Fila de mensagens para comunicação assíncrona entre serviços.

**20-sns.yaml** - Tópico SNS:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: SNSTopic
spec:
  topicName: helm-test-notifications
  displayName: "Notificações de Teste Helm"
```
Pub/Sub para notificações (email, SMS, Lambda, SQS, HTTP).

#### Computação (21-24)

**21-ec2-instance.yaml** - Máquina Virtual EC2:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EC2Instance
spec:
  imageID: ami-12345678               # AMI da região
  instanceType: t2.micro              # Tipo da instância
  subnetID: "subnet-xxx"
  securityGroupIDs: []                # Security Groups
```
Máquina virtual para workloads gerais, aplicações, servidores.

**22-lambda.yaml** - Função Lambda:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: LambdaFunction
spec:
  functionName: helm-test-function
  runtime: python3.9                  # Ambiente de execução
  handler: lambda_function.handler    # Ponto de entrada
  role: arn:aws:iam::xxx:role/lambda-role
  code:
    zipFile: <base64>                 # Código em ZIP base64
  timeout: 30                         # Timeout em segundos
  memorySize: 128                     # MB
```
Função serverless para executar código sem gerenciar servidores.

**23-ecs-cluster.yaml** - Cluster ECS:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: ECSCluster
spec:
  clusterName: helm-test-ecs
```
Cluster para orquestrar containers Docker com Fargate ou EC2.

**24-eks-cluster.yaml** - Cluster EKS Kubernetes:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: EKSCluster
spec:
  clusterName: helm-test-eks
  version: "1.28"                     # Versão do Kubernetes
  roleARN: arn:aws:iam::xxx:role/eks-cluster-role
  resourcesVpcConfig:
    subnetIDs: ["subnet-xxx"]
    endpointPublicAccess: true
    endpointPrivateAccess: false
```
Cluster Kubernetes gerenciado pela AWS.

#### API & CDN (25-26)

**25-api-gateway.yaml** - API Gateway:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: APIGateway
spec:
  name: helm-test-api
  description: "API Gateway de teste Helm"
  protocolType: HTTP                  # HTTP, REST ou WebSocket
```
Gateway para criar, publicar e gerenciar APIs REST/HTTP/WebSocket.

**26-cloudfront.yaml** - CDN CloudFront:
```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: CloudFront
spec:
  comment: "CloudFront de teste Helm"
  enabled: true
  origins:
    - id: test-origin
      domainName: example.com
      customOriginConfig:
        httpPort: 80
        httpsPort: 443
        originProtocolPolicy: http-only
  defaultCacheBehavior:
    targetOriginId: test-origin
    viewerProtocolPolicy: allow-all
    allowedMethods: [GET, HEAD]
    cachedMethods: [GET, HEAD]
    forwardedValues:
      queryString: false              # Não encaminhar query strings
      cookies:
        forward: none                 # Não encaminhar cookies
```
CDN global para distribuir conteúdo com baixa latência e cache na borda.

### Testar um Serviço Específico

#### 1. VPC (Base de Rede)

**Comando:**

```bash
# Aplicar VPC
kubectl apply -f samples/01-vpc.yaml

# Verificar status
kubectl get vpc production-vpc

# Ver detalhes completos
kubectl describe vpc production-vpc
```

#### 2. Bucket S3 (Armazenamento)

**Comando:**

```bash
# Aplicar S3
kubectl apply -f samples/14-s3.yaml

# Verificar status
kubectl get s3bucket app-data

# Status deve mostrar:
# NAME       BUCKET NAME               READY   AGE
# app-data   mycompany-app-data-prod   True    30s
```

#### 3. DynamoDB (Banco de Dados)

**Comando:**

```bash
# Aplicar DynamoDB
kubectl apply -f samples/16-dynamodb.yaml

# Verificar status
kubectl get dynamodbtable users-table
```

### Testar Stack Completo

Para testar múltiplos serviços de uma vez:

```bash
# Aplicar Rede (VPC, Subnet, IGW, NAT)
kubectl apply -f samples/01-vpc.yaml
kubectl apply -f samples/02-subnet.yaml
kubectl apply -f samples/04-internet-gateway.yaml
kubectl apply -f samples/05-nat-gateway.yaml

# Aguardar recursos ficarem Ready
kubectl wait --for=condition=Ready vpc/production-vpc --timeout=60s
kubectl wait --for=condition=Ready subnet/public-subnet-1a --timeout=60s

# Aplicar Load Balancers
kubectl apply -f samples/08-alb.yaml
kubectl apply -f samples/09-nlb.yaml

# Verificar todos os recursos
kubectl get vpc,subnet,internetgateway,natgateway,alb,nlb
```

## Ordem de Criação Recomendada

Para criar uma infraestrutura AWS completa, siga esta ordem:

### 1. Base de Rede

**Comando:**

```bash
kubectl apply -f samples/01-vpc.yaml
kubectl apply -f samples/02-subnet.yaml
kubectl apply -f samples/03-elastic-ip.yaml
kubectl apply -f samples/04-internet-gateway.yaml
kubectl apply -f samples/05-nat-gateway.yaml
kubectl apply -f samples/06-route-table.yaml
kubectl apply -f samples/07-security-group.yaml
```

### 2. Balanceamento de Carga

**Comando:**

```bash
kubectl apply -f samples/08-alb.yaml
kubectl apply -f samples/09-nlb.yaml
```

### 3. Segurança

**Comando:**

```bash
kubectl apply -f samples/10-iam-role.yaml
kubectl apply -f samples/11-kms-key.yaml
kubectl apply -f samples/12-secrets-manager.yaml
kubectl apply -f samples/13-certificate.yaml
```

### 4. Armazenamento & Banco de Dados

**Comando:**

```bash
kubectl apply -f samples/14-s3.yaml
kubectl apply -f samples/15-ecr-repository.yaml
kubectl apply -f samples/16-dynamodb.yaml
kubectl apply -f samples/17-rds-instance.yaml
kubectl apply -f samples/18-elasticache.yaml
```

### 5. Mensageria

**Comando:**

```bash
kubectl apply -f samples/19-sqs.yaml
kubectl apply -f samples/20-sns.yaml
```

### 6. Computação

**Comando:**

```bash
kubectl apply -f samples/21-ec2-instance.yaml
kubectl apply -f samples/22-lambda.yaml
kubectl apply -f samples/23-ecs-cluster.yaml
kubectl apply -f samples/24-eks-cluster.yaml
```

### 7. API & CDN

**Comando:**

```bash
kubectl apply -f samples/25-api-gateway.yaml
kubectl apply -f samples/26-cloudfront.yaml
```

## Verificar Todos os Recursos

**Comando:**

```bash
# Ver todos os recursos criados
kubectl get awsprovider,vpc,subnet,elasticip,internetgateway,natgateway,routetable,securitygroup,alb,nlb

# Ver recursos de segurança
kubectl get iamrole,kmskey,secretsmanagersecret,certificate

# Ver armazenamento e banco de dados
kubectl get s3bucket,ecrrepository,dynamodbtable,rdsinstance,elasticachecluster

# Ver mensageria
kubectl get sqsqueue,snstopic

# Ver computação
kubectl get ec2instance,lambdafunction,ecscluster,ekscluster

# Ver API e CDN
kubectl get apigateway,cloudfront
```

## Comandos Úteis

### Verificar Status do Recurso

**Comando:**

```bash
# Ver status Ready
kubectl get vpc production-vpc -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'

# Ver ID do Recurso AWS
kubectl get vpc production-vpc -o jsonpath='{.status.vpcId}'

# Ver todas as condições de status
kubectl get vpc production-vpc -o jsonpath='{.status.conditions}' | jq
```

### Deletar Recursos

**Comando:**

```bash
# Deletar um recurso específico
kubectl delete vpc production-vpc

# Deletar todos os recursos de um tipo
kubectl delete vpc --all

# Deletar todos os samples
kubectl delete -f samples/
```

:::warning

Por padrão, deletar um CR também **deleta o recurso AWS**. Use `deletionPolicy: Retain` para manter o recurso AWS.

:::

## Desinstalar

**Comando:**

```bash
# Deletar todos os recursos criados
kubectl delete -f samples/

# Desinstalar Helm chart
helm uninstall infra-operator -n iop-system

# Deletar CRDs (opcional)
kubectl delete crd $(kubectl get crd | grep aws-infra-operator.runner.codes | awk '{print $1}')

# Deletar namespace
kubectl delete namespace iop-system
```

## Próximos Passos

- [Rede](/services/networking/vpc)
- [Armazenamento](/services/storage/s3)
- [Computação](/services/compute/ec2)
- [Todos os Serviços](/services/networking/vpc)

## Solução de Problemas

### Pods não iniciam

**Comando:**

```bash
# Ver logs do operador
kubectl logs -n iop-system deploy/infra-operator --tail=100

# Ver eventos
kubectl get events -n iop-system --sort-by='.lastTimestamp'
```

### AWSProvider não está Ready

**Comando:**

```bash
# Verificar credenciais
kubectl describe awsprovider aws-provider

# Testar credenciais manualmente
aws sts get-caller-identity --region us-east-1
```

### Recursos presos em "NotReady"

**Comando:**

```bash
# Ver motivo
kubectl describe vpc production-vpc

# Ver eventos
kubectl get events --field-selector involvedObject.name=production-vpc
```

:::tip

Para mais detalhes de solução de problemas, consulte a documentação específica de cada serviço.

:::
