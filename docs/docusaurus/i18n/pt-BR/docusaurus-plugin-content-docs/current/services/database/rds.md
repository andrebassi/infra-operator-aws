---
title: 'RDS Instance - Banco de Dados Relacional'
description: 'Bancos de dados relacionais gerenciados (PostgreSQL, MySQL, MariaDB, SQL Server, Oracle)'
sidebar_position: 2
---

Crie e gerencie bancos de dados relacionais totalmente gerenciados e escaláveis na AWS com PostgreSQL, MySQL, MariaDB, SQL Server ou Oracle.

## Pré-requisito: Configuração do AWSProvider

Antes de criar qualquer recurso AWS, você precisa configurar um **AWSProvider** que gerencia credenciais e autenticação com a AWS.

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

**IAM Policy - RDS (rds-policy.json):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "rds:CreateDBInstance",
        "rds:DeleteDBInstance",
        "rds:DescribeDBInstances",
        "rds:ModifyDBInstance",
        "rds:StartDBInstance",
        "rds:StopDBInstance",
        "rds:CreateDBSnapshot",
        "rds:DeleteDBSnapshot",
        "rds:AddTagsToResource",
        "rds:RemoveTagsFromResource",
        "rds:ListTagsForResource",
        "rds:CreateDBParameterGroup",
        "rds:ModifyDBParameterGroup"
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
  --role-name infra-operator-rds-role \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for Infra Operator RDS management"

# 4. Criar e anexar policy
aws iam put-role-policy \
  --role-name infra-operator-rds-role \
  --policy-name RDSManagement \
  --policy-document file://rds-policy.json

# 5. Obter Role ARN
aws iam get-role \
  --role-name infra-operator-rds-role \
  --query 'Role.Arn' \
  --output text
```

**Anotar ServiceAccount do Operator:**

```bash
# Adicionar annotation ao ServiceAccount do operator
kubectl annotate serviceaccount infra-operator-controller-manager \
  -n infra-operator-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/infra-operator-rds-role
```
:::note

Substitua `123456789012` pelo seu AWS Account ID e `EXAMPLED539D4633E53DE1B71EXAMPLE` pelo seu OIDC provider ID.

:::

## Visão Geral

Amazon RDS (Relational Database Service) é um serviço de banco de dados relacional totalmente gerenciado que oferece:

**Recursos:**
- **Totalmente Gerenciado**: A AWS gerencia backups, patches, replicação e failover
- **Alta Disponibilidade**: Multi-AZ automático com failover em minutos
- **Múltiplos Engines**: PostgreSQL, MySQL, MariaDB, SQL Server, Oracle
- **Escalabilidade**: Aumente ou diminua a capacidade conforme necessário
- **Backups Automatizados**: Retenção configurável (até 35 dias)
- **Point-in-Time Recovery (PITR)**: Restaure para qualquer ponto nos últimos 35 dias
- **Criptografia em Repouso**: AES-256 com AWS KMS
- **Criptografia em Trânsito**: SSL/TLS automático
- **Performance Insights**: Monitore e otimize performance
- **Enhanced Monitoring**: Métricas detalhadas de OS, IO, CPU
- **Read Replicas**: Replicação de leitura entre regiões (síncrona ou assíncrona)
- **Patching Automatizado**: Janelas de manutenção configuráveis
- **Parameter Groups**: Configuração customizada do banco de dados
- **Security Groups**: Controle de acesso em nível de rede

**Status**: ⚠️ Requer LocalStack Pro ou AWS Real

## Início Rápido

**RDS PostgreSQL:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: e2e-postgres-db
  namespace: default
spec:
  providerRef:
    name: localstack

  # Identificador único do banco de dados
  dbInstanceIdentifier: e2e-test-postgres

  # Database engine
  engine: postgres
  engineVersion: "14.7"

  # Instance class
  dbInstanceClass: db.t3.micro

  # Storage
  allocatedStorage: 20

  # Credenciais do administrador
  masterUsername: dbadmin
  masterUserPassword: Test123456!

  # Nome do banco de dados inicial
  dbName: testdb
  port: 5432

  # Configuração
  multiAZ: false
  publiclyAccessible: true
  storageEncrypted: true

  # Backups
  backupRetentionPeriod: 7
  preferredBackupWindow: "03:00-04:00"

  # Tags
  tags:
    Environment: test
    ManagedBy: infra-operator
    Database: postgres

  deletionPolicy: Delete
```

**RDS MySQL:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: e2e-mysql-db
  namespace: default
spec:
  providerRef:
    name: localstack

  # Identificador único
  dbInstanceIdentifier: e2e-test-mysql

  # MySQL engine
  engine: mysql
  engineVersion: "8.0.33"

  # Instance class
  dbInstanceClass: db.t3.small

  # Storage
  allocatedStorage: 30

  # Credenciais
  masterUsername: admin
  masterUserPassword: MyPassword123!

  # Banco de dados inicial
  dbName: mydb
  port: 3306

  # Multi-AZ habilitado
  multiAZ: true
  publiclyAccessible: false
  storageEncrypted: true

  # Backups
  backupRetentionPeriod: 14

  # Tags
  tags:
    Environment: test
    ManagedBy: infra-operator
    Database: mysql

  deletionPolicy: Delete
```

**RDS PostgreSQL Produção:**

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: app-database
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Identificador único do banco de dados (2-63 caracteres)
  dbInstanceIdentifier: app-db-prod

  # Database engine
  engine: postgres
  engineVersion: "15.4"

  # Instance class (capacidade)
  dbInstanceClass: db.t3.micro

  # Storage
  allocatedStorage: 20
  storageType: gp3
  iops: 3000

  # Credenciais do administrador
  masterUsername: postgres
  masterUserPasswordSecretRef:
    name: db-password
    key: password

  # Multi-AZ para alta disponibilidade
  multiAZ: true

  # VPC e segurança
  dbSubnetGroupName: private-subnet-group
  vpcSecurityGroupRefs:
    - name: db-sg

  # Backups
  backupRetentionPeriod: 7
  preferredBackupWindow: "03:00-04:00"
  preferredMaintenanceWindow: "sun:04:00-sun:05:00"

  # Proteção contra deleção
  deletionProtection: true

  # Tagging
  tags:
    Environment: production
    Application: web-app

  deletionPolicy: Retain
```

**Aplicar:**

```bash
kubectl apply -f rds.yaml
```

**Verificar Status:**

```bash
kubectl get rdsinstances
kubectl describe rdsinstance e2e-postgres-db
kubectl get rdsinstance e2e-postgres-db -o yaml
```
## Referência de Configuração

### Campos Obrigatórios

Referência ao recurso AWSProvider para autenticação

  Nome do recurso AWSProvider

Identificador único para a instância RDS (2 a 63 caracteres)

  **Regras:**
  - Apenas caracteres alfanuméricos e hífens
  - Não pode começar ou terminar com hífen
  - Deve ser único por região
  - Não pode ser modificado após criação

  **Exemplo:**

  ```yaml
  dbInstanceIdentifier: myapp-db-prod
  ```

Database engine

  **Opções:**
  - `postgres` - PostgreSQL
  - `mysql` - MySQL
  - `mariadb` - MariaDB
  - `sqlserver-ex` - SQL Server Express
  - `sqlserver-se` - SQL Server Standard
  - `sqlserver-ee` - SQL Server Enterprise
  - `oracle-se2` - Oracle Standard Edition 2
  - `oracle-ee` - Oracle Enterprise Edition

  **Exemplo:**

  ```yaml
  engine: postgres
  ```

Versão do database engine

  **Exemplos:**
  - PostgreSQL: `15.4`, `14.9`, `13.13`
  - MySQL: `8.0.35`, `8.0.34`, `5.7.44`
  - MariaDB: `10.6.14`, `10.5.21`, `10.4.32`

  **Exemplo:**

  ```yaml
  engineVersion: "15.4"
  ```

  **Nota:** Sempre especifique versões com patch (ex: 15.4, não 15)

Instance class (CPU, RAM, performance)

  **Família db.t3 (Burstable - Dev/Test):**
  - `db.t3.micro` - 1 vCPU, 1 GB RAM (~$0.017/hora)
  - `db.t3.small` - 1 vCPU, 2 GB RAM (~$0.034/hora)
  - `db.t3.medium` - 1 vCPU, 4 GB RAM (~$0.068/hora)

  **Família db.t4g (Graviton - mais barato):**
  - `db.t4g.micro` - 1 vCPU, 1 GB RAM
  - `db.t4g.small` - 1 vCPU, 2 GB RAM

  **Família db.m5 (General Purpose):**
  - `db.m5.large` - 2 vCPU, 8 GB RAM (~$0.175/hora)
  - `db.m5.xlarge` - 4 vCPU, 16 GB RAM (~$0.350/hora)
  - `db.m5.2xlarge` - 8 vCPU, 32 GB RAM (~$0.700/hora)

  **Família db.r5 (Memory Optimized - Cache/Analytics):**
  - `db.r5.large` - 2 vCPU, 16 GB RAM (~$0.280/hora)
  - `db.r5.xlarge` - 4 vCPU, 32 GB RAM (~$0.560/hora)

  **Exemplo:**

  ```yaml
  dbInstanceClass: db.t3.micro
  ```

Espaço de armazenamento alocado em GB

  **Range:** 20 - 65,536 GB (depende do engine)

  **Tipos de armazenamento:**
  - **gp2** (General Purpose): 20-65,536 GB (padrão)
  - **gp3** (General Purpose v3): 20-65,536 GB (recomendado, melhor performance)
  - **io1** (Provisioned IOPS): 100-65,536 GB (alta performance)
  - **io2** (Provisioned IOPS v2): 100-65,536 GB (máxima performance)

  **Exemplo:**

  ```yaml
  allocatedStorage: 20
  storageType: gp3
  ```

  **Nota:** Você pode aumentar o tamanho depois, mas não pode diminuir

Nome de usuário do administrador do banco de dados

  **Regras:**
  - 1 a 16 caracteres (depende do engine)
  - Apenas alfanuméricos
  - Não pode ser `admin`, `root`, `postgres` (alguns engines)
  - Não pode começar com número

  **Exemplo:**

  ```yaml
  masterUsername: dbadmin
  ```

Referência ao Secret do Kubernetes contendo a senha

  Nome do Secret do Kubernetes

Chave dentro do Secret

**Exemplo de Secret:**
  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: db-password
    namespace: default
  type: Opaque
  stringData:
    password: SuperSecurePassword123!
  ```

  **Regras de Senha:**
  - Mínimo 8 caracteres
  - Contém: letras, números, símbolos especiais
  - Não pode conter `@`, `/`, `"`, ou `\`

### Campos Opcionais - Storage

Tipo de armazenamento EBS

  **Opções:**
  - `gp2`: General Purpose SSD (padrão, bom para a maioria)
  - `gp3`: General Purpose SSD v3 (recomendado, melhor performance)
  - `io1`: Provisioned IOPS SSD (performance previsível)
  - `io2`: Provisioned IOPS SSD v2 (máxima performance)

  **Exemplo:**

  ```yaml
  storageType: gp3
  ```

IOPS provisionados (apenas io1/io2)

  **Range:** 1,000 - 64,000 IOPS

  **Nota:** Para gp3, IOPS é configurado separadamente do tamanho:

  ```yaml
  storageType: io1
  allocatedStorage: 100
  iops: 5000
  ```

Throughput de armazenamento para gp3 (MB/s)

  **Range:** 125 - 1,000 MB/s:

  ```yaml
  storageType: gp3
  storageThroughput: 500
  ```

### Campos Opcionais - Rede e Segurança

Nome do DB Subnet Group (subnets privadas)

  **Importante:** DEVE existir previamente na AWS:

  ```yaml
  dbSubnetGroupName: private-subnet-group
  ```

  **Criar subnet group via AWS CLI:**
  ```bash
  aws rds create-db-subnet-group \
    --db-subnet-group-name private-subnet-group \
    --db-subnet-group-description "Private subnets for RDS" \
    --subnet-ids subnet-xxx subnet-yyy
  ```

Referências aos Security Groups do Kubernetes para controle de acesso

  Nome do recurso SecurityGroup no Kubernetes

**Exemplo:**

```yaml
  vpcSecurityGroupRefs:
    - name: rds-security-group
  ```

  **Alternativa - usar IDs diretos:**
  ```yaml
  vpcSecurityGroupIds:
    - sg-0123456789abcdef0
    - sg-0123456789abcdef1
  ```

Se o banco de dados é acessível via internet pública

  **⚠️ Segurança:** Nunca habilite em produção!:

  ```yaml
  publiclyAccessible: false  # Manter privado
  ```

Protege contra deleção acidental

  **Recomendado:** true em produção:

  ```yaml
  deletionProtection: true
  ```

### Campos Opcionais - Backups e Recuperação

Dias de retenção de backup automatizado

  **Range:** 1 - 35 dias

  **Recomendações:**
  - Desenvolvimento: 1-7 dias
  - Staging: 7-14 dias
  - Produção: 14-35 dias

  **Exemplo:**

  ```yaml
  backupRetentionPeriod: 30
  ```

  **Custo:** Aumenta com a retenção

Janela diária de backup (UTC, formato HH:MM-HH:MM)

  **Padrão:** Selecionado automaticamente:

  ```yaml
  preferredBackupWindow: "03:00-04:00"  # 3AM-4AM UTC
  ```

  **Dica:** Escolha horário de baixo uso

Janela semanal de manutenção para patches (formato: ddd:HH:MM-ddd:HH:MM)

  **Padrão:** Selecionado automaticamente:

  ```yaml
  preferredMaintenanceWindow: "sun:04:00-sun:05:00"
  # Domingo, 4AM-5AM UTC
  ```

  **Dias válidos:** mon, tue, wed, thu, fri, sat, sun

Pular criação de snapshot final ao deletar banco de dados

  **Cuidado:** true pode perder dados!:

  ```yaml
  skipFinalSnapshot: false  # Sempre criar snapshot final
  ```

Identificador do snapshot final ao deletar

  **Exemplo:**

  ```yaml
  finalDBSnapshotIdentifier: app-db-final-snapshot-2025-11-22
  ```

### Campos Opcionais - Alta Disponibilidade e Replicação

Habilitar Multi-AZ (alta disponibilidade)

  **Benefícios:**
  - Failover automático em caso de falha (minutos)
  - Replicação síncrona para integridade de dados
  - ~50% de aumento de custo

  **Recomendado:** true em produção:

  ```yaml
  multiAZ: true
  ```

Usar IAM para autenticação (ao invés de senha)

  **Exemplo:**

  ```yaml
  enableIAMDatabaseAuthentication: true
  ```

  **Benefícios:**
  - Sem senha hardcoded
  - Tokens temporários
  - Auditoria automática

### Campos Opcionais - Criptografia

Habilitar criptografia em repouso

  **Padrão:** true (recomendado):

  ```yaml
  storageEncrypted: true
  ```

ARN da chave KMS da AWS para criptografia

  **Padrão:** Chave gerenciada pela AWS (sem custo):

  ```yaml
  storageEncrypted: true
  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012
  ```

### Campos Opcionais - Monitoramento e Otimização

Habilitar Performance Insights (análise detalhada)

  **Exemplo:**

  ```yaml
  enablePerformanceInsights: true
  ```

  **Custo:** ~$0.02/hora adicional

Habilitar Enhanced Monitoring (métricas do OS)

  **Exemplo:**

  ```yaml
  enableEnhancedMonitoring: true
  monitoringInterval: 60  # A cada 60 segundos
  ```

  **Custo:** Baseado no intervalo

Intervalo do Enhanced Monitoring (segundos)

  **Opções:** 0 (desabilitado), 1, 5, 10, 15, 30, 60:

  ```yaml
  monitoringInterval: 60
  ```

### Campos Opcionais - Outros

Pares chave-valor para organização e billing

  **Exemplo:**

  ```yaml
  tags:
    Environment: production
    Application: web-app
    Team: backend
    CostCenter: engineering
    ManagedBy: infra-operator
  ```

O que acontece com a instância quando o CR é deletado

  **Opções:**
  - `Delete`: Instância é deletada da AWS
  - `Retain`: Instância permanece na AWS mas não gerenciada
  - `Orphan`: Remove apenas gerenciamento

  **Exemplo:**

  ```yaml
  deletionPolicy: Retain  # Para produção
  ```

## Campos de Status

Após a instância ser criada, os seguintes campos de status são populados:

ARN completo da instância RDS

  ```
  arn:aws:rds:us-east-1:123456789012:db:app-db-prod
  ```

Informações de conexão do banco de dados

  Hostname do banco de dados para conexão

      ```
      app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com
      ```

Porta de conexão (padrão: 5432 PostgreSQL, 3306 MySQL)


Estado atual da instância

  **Valores possíveis:**
  - `creating` - Criando
  - `available` - Disponível e pronto
  - `deleting` - Sendo deletado
  - `modifying` - Configuração sendo alterada
  - `backing-up` - Backup em andamento
  - `maintenance` - Manutenção programada
  - `failed` - Erro na criação

Espaço alocado em GB

Versão do engine em execução

Se Multi-AZ está habilitado

`true` quando a instância está `available` e pronta

Timestamp da última sincronização com AWS

## Exemplos

### PostgreSQL RDS de Produção com Multi-AZ

Banco de dados para aplicação web com alta disponibilidade:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: production-postgres
  namespace: default
spec:
  providerRef:
    name: production-aws

  # Identificação
  dbInstanceIdentifier: myapp-db-prod
  engine: postgres
  engineVersion: "15.4"

  # Classe e storage
  dbInstanceClass: db.t3.small
  allocatedStorage: 50
  storageType: gp3
  iops: 3000
  storageThroughput: 250

  # Credenciais
  masterUsername: postgres
  masterUserPasswordSecretRef:
    name: db-password
    key: password

  # Rede
  dbSubnetGroupName: private-subnet-group
  vpcSecurityGroupRefs:
    - name: rds-security-group
  publiclyAccessible: false

  # Alta Disponibilidade
  multiAZ: true
  deletionProtection: true

  # Backups
  backupRetentionPeriod: 30
  preferredBackupWindow: "03:00-04:00"
  preferredMaintenanceWindow: "sun:04:00-sun:05:00"

  # Monitoramento
  enablePerformanceInsights: true

  # Segurança
  storageEncrypted: true

  # Tags
  tags:
    Environment: production
    Application: myapp
    Team: backend
    CostCenter: engineering

  deletionPolicy: Retain
```

### RDS MySQL para Desenvolvimento

Banco de dados simples para desenvolvimento local:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: dev-mysql
  namespace: default
spec:
  providerRef:
    name: dev-aws

  dbInstanceIdentifier: myapp-db-dev

  engine: mysql
  engineVersion: "8.0.35"

  # Instância pequena para dev
  dbInstanceClass: db.t3.micro
  allocatedStorage: 20
  storageType: gp3

  masterUsername: admin
  masterUserPasswordSecretRef:
    name: dev-db-password
    key: password

  # Dev não precisa de Multi-AZ
  multiAZ: false
  deletionProtection: false

  # Backups menos frequentes
  backupRetentionPeriod: 7

  tags:
    Environment: development
    Application: myapp

  deletionPolicy: Delete
```

### RDS com Read Replica

Banco de dados primário com replicação de leitura para analytics:

```yaml
# Banco de dados primário
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: primary-db
  namespace: default
spec:
  providerRef:
    name: production-aws

  dbInstanceIdentifier: myapp-db-primary
  engine: postgres
  engineVersion: "15.4"

  dbInstanceClass: db.m5.large
  allocatedStorage: 100
  storageType: gp3

  masterUsername: postgres
  masterUserPasswordSecretRef:
    name: db-password
    key: password

  multiAZ: true
  backupRetentionPeriod: 35

  # Habilitar backups (necessário para réplicas)
  backupRetentionPeriod: 7

  tags:
    Environment: production
    Role: primary

  deletionPolicy: Retain

---
# Read Replica (para analytics/reporting)
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: read-replica-db
  namespace: default
spec:
  providerRef:
    name: production-aws

  dbInstanceIdentifier: myapp-db-read-replica

  # Criar a partir do banco de dados primário
  replicateSourceDb: myapp-db-primary

  # Mesmo engine, pode ser classe menor
  dbInstanceClass: db.t3.small

  # Não precisa configurar senha (herda do primário)
  # Réplica em AZ diferente para HA
  availabilityZone: us-east-1b

  tags:
    Environment: production
    Role: read-replica

  deletionPolicy: Delete
```

### RDS com Criptografia e PITR Completo

Banco de dados crítico com máxima segurança e recuperação:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: critical-database
  namespace: default
spec:
  providerRef:
    name: production-aws

  dbInstanceIdentifier: critical-db-prod

  engine: postgres
  engineVersion: "15.4"

  dbInstanceClass: db.r5.xlarge  # Memory optimized
  allocatedStorage: 500
  storageType: io2
  iops: 20000

  masterUsername: postgres
  masterUserPasswordSecretRef:
    name: critical-db-password
    key: password

  # VPC privada com segurança
  dbSubnetGroupName: critical-subnet-group
  vpcSecurityGroupRefs:
    - name: critical-rds-sg
  publiclyAccessible: false

  # HA e proteção
  multiAZ: true
  deletionProtection: true

  # Backups com PITR completo
  backupRetentionPeriod: 35
  preferredBackupWindow: "02:00-03:00"
  skipFinalSnapshot: false
  finalDBSnapshotIdentifier: critical-db-final-backup

  # Criptografia com KMS
  storageEncrypted: true
  kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee

  # Monitoramento completo
  enablePerformanceInsights: true
  enableEnhancedMonitoring: true
  monitoringInterval: 1

  # Autenticação IAM
  enableIAMDatabaseAuthentication: true

  # Manutenção
  preferredMaintenanceWindow: "sun:03:00-sun:04:00"

  tags:
    Environment: production
    CriticalData: "true"
    BackupRequired: "true"
    Compliance: "required"

  deletionPolicy: Retain
```

### RDS MariaDB para WordPress/CMS

Banco de dados para aplicação web tradicional:

```yaml
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: RDSInstance
metadata:
  name: wordpress-db
  namespace: default
spec:
  providerRef:
    name: production-aws

  dbInstanceIdentifier: wordpress-db

  engine: mariadb
  engineVersion: "10.6.14"

  dbInstanceClass: db.t3.small
  allocatedStorage: 50
  storageType: gp3

  masterUsername: wordpress
  masterUserPasswordSecretRef:
    name: wordpress-db-password
    key: password

  dbSubnetGroupName: web-subnet-group
  vpcSecurityGroupRefs:
    - name: wordpress-rds-sg

  multiAZ: true
  backupRetentionPeriod: 14

  tags:
    Environment: production
    Application: wordpress

  deletionPolicy: Retain
```

## Verificação

### Verificar Status via kubectl

**Comando:**

```bash
# Listar todas as instâncias RDS
kubectl get rdsinstances

# Obter informações detalhadas
kubectl get rdsinstance production-postgres -o yaml

# Monitorar criação em tempo real
kubectl get rdsinstance production-postgres -w

# Ver eventos e status
kubectl describe rdsinstance production-postgres
```

### Verificar na AWS

**AWS CLI:**

```bash
# Listar instâncias RDS
aws rds describe-db-instances \
  --query 'DBInstances[].{Identifier:DBInstanceIdentifier,Status:DBInstanceStatus,Engine:Engine,Class:DBInstanceClass}' \
  --output table

# Obter detalhes completos
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --output json | jq '.DBInstances[0]'

# Ver endpoint de conexão
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].Endpoint'

# Testar conexão (PostgreSQL)
psql -h app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com \
  -U postgres \
  -d postgres

# Ver backups
aws rds describe-db-snapshots \
  --db-instance-identifier app-db-prod

# Ver status multi-AZ
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].MultiAZ'

```

**LocalStack:**

```bash
# Para testes com LocalStack
export AWS_ENDPOINT_URL=http://localhost:4566

aws rds describe-db-instances

aws rds describe-db-instances \
  --db-instance-identifier app-db-prod
```

**psql (PostgreSQL):**

```bash
# Conectar ao banco de dados
psql -h <endpoint> -U postgres -d postgres

# Ver usuários
\du

# Ver bancos de dados
\l

# Ver espaço usado
SELECT pg_database.datname,
       pg_size_pretty(pg_database_size(pg_database.datname))
FROM pg_database;

# Desconectar
\q
```

**mysql (MySQL/MariaDB):**

```bash
# Conectar ao banco de dados
mysql -h <endpoint> -u admin -p

# Ver bancos de dados
SHOW DATABASES;

# Ver usuários
SELECT user, host FROM mysql.user;

# Ver espaço usado
SELECT table_schema,
       ROUND(SUM(data_length+index_length)/1024/1024,2) AS size_mb
FROM information_schema.tables
GROUP BY table_schema;

# Sair
EXIT;
```

### Output Esperado

**Exemplo:**

```yaml
status:
  dbInstanceArn: arn:aws:rds:us-east-1:123456789012:db:app-db-prod
  dbInstanceStatus: available
  endpoint:
    address: app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com
    port: 5432
  engine: postgres
  engineVersion: "15.4"
  dbInstanceClass: db.t3.small
  allocatedStorage: 50
  multiAZ: true
  storageEncrypted: true
  ready: true
  lastSyncTime: "2025-11-22T20:45:22Z"
```

## Solução de Problemas

### RDS travado em creating por mais de 30 minutos

**Sintomas:** `dbInstanceStatus: creating` indefinidamente

**Causas comuns:**
1. Subnet group não existe ou é inválido
2. Security group não encontrado
3. Cota de instâncias RDS atingida
4. Problema de conectividade com AWS

**Soluções:**
```bash
# Verificar status detalhado
kubectl describe rdsinstance app-database

# Ver logs do operator
kubectl logs -n infra-operator-system \
  deploy/infra-operator-controller-manager \
  --tail=100 | grep -i rds

# Verificar se subnet group existe
aws rds describe-db-subnet-groups \
  --db-subnet-group-name private-subnet-group

# Verificar se AWSProvider está ready
kubectl get awsprovider
kubectl describe awsprovider production-aws

# Forçar sincronização
kubectl annotate rdsinstance app-database \
  force-sync="$(date +%s)" --overwrite

# Último recurso: deletar e recriar (com Retain)
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"deletionPolicy":"Retain"}}'
```

### Timeout de conexão ao conectar no banco de dados

**Sintomas:** `psql: could not translate host name` ou `Connection refused`

**Causas comuns:**
1. Security group não permite conexão (porta bloqueada)
2. Banco de dados não é publicamente acessível e está conectando de fora da VPC
3. Hostname/endpoint incorreto
4. Rede não roteada corretamente

**Soluções:**
```bash
# Obter endpoint correto
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].Endpoint'

# Verificar se security group permite ingress
aws ec2 describe-security-groups \
  --group-ids sg-0123456789abcdef0 \
  --query 'SecurityGroups[0].IpPermissions'

# DEVE ter regra como:
# IpProtocol: tcp, FromPort: 5432, ToPort: 5432
# CidrIp: 10.0.0.0/16 (sua VPC)

# Se conectando de fora da VPC:
# 1. Habilitar publiclyAccessible (não recomendado)
# 2. Ou usar bastion/jump host
# 3. Ou usar VPN

# Testar com telnet/nc
nc -zv app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com 5432

# Testar com psql verbose
psql -h app-db-prod.c9akciq32.us-east-1.rds.amazonaws.com \
  -U postgres \
  -d postgres \
  -v

# Se usando Multi-AZ, failover pode estar em andamento
# Tente novamente em 5 minutos
```

### Espaço em disco esgotado (storage full)

**Sintomas:** Erro `Disk full` ao executar queries, aplicação lenta

**Causas:**
1. Dados cresceram além do esperado
2. Logs acumulados ou backups
3. Falta de limpeza de dados antigos

**Soluções:**
```bash
# Ver espaço disponível
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].AllocatedStorage'

# Aumentar storage (pode levar minutos)
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"allocatedStorage":100}}'

# Ver progresso
kubectl get rdsinstance app-database -w

# Se PostgreSQL, limpar espaço
# Conectar ao banco de dados:
psql -h <endpoint> -U postgres -d postgres

# Vacuum full (recuperar espaço)
VACUUM FULL;
VACUUM ANALYZE;

# Se MySQL, otimizar tabelas
mysql -h <endpoint> -u admin -p

OPTIMIZE TABLE table_name;

# Ver tamanho do banco de dados
SELECT pg_database.datname,
       pg_size_pretty(pg_database_size(pg_database.datname))
FROM pg_database;
```

### Custos altos de RDS

**Sintomas:** Conta AWS com cobranças inesperadas de RDS

**Causas comuns:**
1. Classe de instância muito grande
2. IOPS provisionados altos
3. Muitos backups retidos
4. Replicação cross-region

**Soluções:**
```bash
# Calcular custo atual
# Use AWS Pricing Calculator
# t3.small: ~$0.034/hora * 730 = ~$25/mês
# gp3 storage: ~$0.12/GB/mês

# Reduzir classe (se possível)
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"dbInstanceClass":"db.t3.micro"}}'

# Reduzir retenção de backup
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"backupRetentionPeriod":7}}'

# Deletar snapshots antigos desnecessários
aws rds delete-db-snapshot \
  --db-snapshot-identifier app-db-snapshot-old

# Para dev, deletar após uso
kubectl delete rdsinstance dev-database
```

### Backup falha ou demora muito

**Sintomas:** Status de backup travado em `backing-up` por horas

**Causas:**
1. Banco de dados muito grande
2. IO saturado durante backup
3. Muitas transações durante backup

**Soluções:**
```bash
# Ver status de backup
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].LatestRestorableTime'

# Ver snapshots completos
aws rds describe-db-snapshots \
  --db-instance-identifier app-db-prod

# Aumentar janela de backup
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"preferredBackupWindow":"02:00-04:00"}}'

# Aumentar IOPS para performance de backup
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"iops":5000}}'

# Criar snapshot manual
aws rds create-db-snapshot \
  --db-instance-identifier app-db-prod \
  --db-snapshot-identifier manual-backup-$(date +%Y%m%d-%H%M%S)
```

### Failover Multi-AZ lento ou falha

**Sintomas:** Após falha, aplicação offline por 5+ minutos

**Causa:** Failover automático pode demorar, especialmente se Multi-AZ não configurado corretamente

**Soluções:**
```bash
# Verificar se Multi-AZ está habilitado
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].MultiAZ'

# Habilitar se não estiver
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"multiAZ":true}}'

# Testar failover manualmente
# CUIDADO: Causa downtime!
aws rds reboot-db-instance \
  --db-instance-identifier app-db-prod \
  --force-failover

# Ver progresso de reboot
aws rds describe-db-instances \
  --db-instance-identifier app-db-prod \
  --query 'DBInstances[0].DBInstanceStatus'

# Implementar retry na aplicação
# Usar connection pooling
# Configurar failover no nível da aplicação
```

### Read Replica não sincronizando ou com lag

**Sintomas:** Réplica mostra dados desatualizados, não consegue conectar

**Causas:**
1. Lag de replicação (atraso de rede)
2. Banco de dados primário sob pressão de IO
3. Problemas de rede

**Soluções:**
```bash
# Ver lag de réplica
aws rds describe-db-instances \
  --db-instance-identifier myapp-db-read-replica \
  --query 'DBInstances[0].StatusInfos'

# Aumentar classe de réplica se estiver subprovisionada
kubectl patch rdsinstance read-replica-db \
  --type merge \
  -p '{"spec":{"dbInstanceClass":"db.t3.small"}}'

# Aumentar IOPS no primário
kubectl patch rdsinstance primary-db \
  --type merge \
  -p '{"spec":{"iops":5000}}'

```

### Performance degradada (queries lentas)

**Sintomas:** Queries normais ficam lentas, CPU/IO alto

**Causas:**
1. Falta de índices
2. Plano de query subótimo
3. Falta de memória/CPU
4. Locks de transações

**Soluções:**
```bash
# Habilitar Performance Insights
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"enablePerformanceInsights":true}}'

# Usar console AWS Performance Insights para análise

# Aumentar memória/CPU se necessário
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"dbInstanceClass":"db.m5.large"}}'

# Conectar e analisar queries
psql -h <endpoint> -U postgres -d postgres

# Ver queries lentas (PostgreSQL)
SELECT query, calls, mean_exec_time, max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

# Ver índices faltando
SELECT schemaname, tablename
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY schemaname, tablename;

# Criar índices apropriados
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_orders_date ON orders(created_at);
```

### Erro ao deletar banco de dados (finalizer travado)

**Sintomas:** `kubectl delete rdsinstance` pendente indefinidamente

**Causa:** Finalizer não consegue deletar snapshot final ou banco de dados

**Soluções:**
```bash
# Ver detalhes
kubectl describe rdsinstance app-database

# Ver finalizers
kubectl get rdsinstance app-database -o yaml | grep finalizers

# Opção 1: Mudar deletionPolicy antes de deletar
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"deletionPolicy":"Retain"}}'

# Então deletar
kubectl delete rdsinstance app-database

# Opção 2: Deletar snapshot final manualmente
aws rds delete-db-snapshot \
  --db-snapshot-identifier app-db-final-backup

# Então forçar delete do CR
kubectl patch rdsinstance app-database \
  -p '{"metadata":{"finalizers":[]}}' \
  --type=merge

# Opção 3: Manter snapshot final, apenas remover CR
kubectl patch rdsinstance app-database \
  --type merge \
  -p '{"spec":{"skipFinalSnapshot":true}}'

kubectl delete rdsinstance app-database
```

## Boas Práticas

:::note Boas Práticas

- **Habilite Multi-AZ para produção** — Failover automático em minutos, ~50% de custo a mais mas essencial para dados críticos, teste failover regularmente
- **Habilite criptografia em todos os lugares** — Criptografia de storage (AES-256), AWS KMS para chaves customizadas, SSL/TLS para conexões, criptografia de backup para compliance
- **Configure retenção de backup apropriada** — 7-35 dias dependendo da criticidade, mínimo 14 dias para produção, PITR até 35 dias, teste restore regularmente
- **Restrinja acesso de security group** — Apenas IPs/SGs necessários, nunca 0.0.0.0/0, segregue dev/staging/prod, audite regras regularmente
- **Use apenas subnets privadas** — Nunca publicamente acessível em produção, use NAT gateway para outbound, bastion/jump host para acesso admin
- **Habilite monitoramento e insights** — Performance Insights para tuning, alertas CloudWatch para anomalias, monitore métricas de CPU/Storage/IOPS
- **Ajuste parameter groups** — Customize configurações por engine (shared_buffers, work_mem para PostgreSQL), documente razão das mudanças, teste em dev primeiro
- **Tagueie todos os recursos** — Environment (dev/staging/prod), application, cost center, owner/team, flag de backup required
- **Dimensione instâncias corretamente** — Comece pequeno e monitore crescimento, use t3/t4g para cargas variáveis, m5 para cargas de produção previsíveis
- **Otimize custos** — Delete dev/staging após uso, retenção de backup apropriada, reserved instances para produção
- **Planeje disaster recovery** — Read replicas para DR regional, snapshots em outra região, documente RTO/RPO, automatize failover
- **Habilite audit e compliance** — CloudTrail para chamadas API, CloudWatch Logs para queries, autenticação IAM de banco de dados
- **Otimize queries** — Performance Insights para bottlenecks, logs de slow query, estratégia de índice apropriada, connection pooling na app

:::

## Casos de Uso

### 1. Aplicação Web Transacional

E-commerce ou SaaS com muitas transações pequenas:

```yaml
dbInstanceClass: db.t3.small
allocatedStorage: 100
multiAZ: true
backupRetentionPeriod: 30
# Índices em user_id, order_id, timestamps
```

### 2. E-commerce com Carrinhos e Pedidos

Dados de alto volume com transações críticas:

```yaml
dbInstanceClass: db.m5.large
allocatedStorage: 500
storageType: io1
iops: 10000
multiAZ: true
backupRetentionPeriod: 35
# Read replica para analytics
```

### 3. CMS (WordPress, Drupal)

Aplicações web com conteúdo dinâmico:

```yaml
engine: mysql  # ou mariadb
dbInstanceClass: db.t3.small
allocatedStorage: 50
multiAZ: true
enableCloudwatchLogsExports: [slowquery, error]
```

### 4. ERP/CRM com Alta Concorrência

Múltiplos usuários acessando simultaneamente:

```yaml
dbInstanceClass: db.m5.2xlarge
allocatedStorage: 1000
storageType: io2
iops: 64000
multiAZ: true
backupRetentionPeriod: 35
enableEnhancedMonitoring: true
```

## Recursos Relacionados

- [VPC e Subnets](/services/networking/vpc)

  - [Security Groups](/services/networking/security-group)

  - [Secrets Manager](/services/security/secrets-manager)

  - [Lambda](/services/compute/lambda)

  - [DynamoDB](/services/database/dynamodb)
---
