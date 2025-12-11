# E2E Test Suite

Suite completa de testes end-to-end para o Infra Operator com LocalStack.

## Pré-requisitos

1. **Kubernetes cluster** rodando (minikube, kind, orbstack, etc)
2. **LocalStack** instalado e rodando no cluster
3. **Infra Operator** deployado no cluster
4. **Samples aplicados** - recursos de teste criados

## Instalação dos Pré-requisitos

### 1. Deploy do LocalStack

\`\`\`bash
helm repo add localstack https://localstack.github.io/helm-charts
helm install localstack localstack/localstack -n localstack --create-namespace
\`\`\`

### 2. Deploy do Infra Operator

\`\`\`bash
# Aplicar CRDs
kubectl apply -f config/crd/bases/

# Aplicar RBAC e Deployment
kubectl apply -f config/manager/namespace.yaml
kubectl apply -f config/rbac/
kubectl apply -f config/manager/deployment.yaml
\`\`\`

### 3. Aplicar Samples de Teste

\`\`\`bash
# AWSProvider
kubectl apply -f config/samples/aws_v1alpha1_awsprovider.yaml

# Storage
kubectl apply -f /tmp/test-samples/storage/01-s3-test.yaml
kubectl apply -f /tmp/test-samples/database/13-dynamodb-test.yaml

# Messaging
kubectl apply -f /tmp/test-samples/messaging/17-sqs-test.yaml
kubectl apply -f /tmp/test-samples/messaging/18-sns-test.yaml

# Networking
kubectl apply -f /tmp/test-samples/networking/01-vpc-test.yaml
kubectl apply -f /tmp/test-samples/networking/02-subnet-test.yaml

# Compute
kubectl apply -f /tmp/test-samples/compute/11-lambda-test.yaml

# Database (vai falhar no LocalStack Free)
kubectl apply -f /tmp/test-samples/database/14-rdsinstance-test.yaml
\`\`\`

## Executando os Testes

### Execução Básica

\`\`\`bash
./test/e2e/e2e_test.sh
\`\`\`

### Com Logging Detalhado

\`\`\`bash
./test/e2e/e2e_test.sh 2>&1 | tee /tmp/e2e-test-results.log
\`\`\`

## Testes Incluídos

### 1. Pre-requisite Tests
- ✅ Operator está rodando em \`iop-system\`
- ✅ LocalStack está rodando em \`localstack\`

### 2. Provider Tests
- ✅ AWSProvider conecta ao LocalStack
- ✅ Account ID = 000000000000

### 3. Storage Tests
- ✅ S3 Bucket criado e READY
- ✅ DynamoDB Table criado e ACTIVE

### 4. Messaging Tests
- ✅ SQS Queue criado com URL válida
- ✅ SNS Topic criado com ARN válida

### 5. Networking Tests
- ✅ VPC criado com VPC-ID e CIDR
- ✅ Subnet criado com Subnet-ID e IPs disponíveis

### 6. Compute Tests
- ⚠️ Lambda Function (pode ter issues no LocalStack Free)

### 7. Database Tests
- 🚫 RDS Instance (não disponível no LocalStack Free)

## Resultados Esperados

\`\`\`
==========================================
  Test Results
==========================================
Passed:  8
Failed:  0
Skipped: 2
==========================================

[PASS] All tests passed!
\`\`\`

## Troubleshooting

### Teste falhou: "Operator pod not found"

Verifique se o operator está rodando:

\`\`\`bash
kubectl get pods -n iop-system
kubectl logs -n iop-system deploy/infra-operator-controller-manager
\`\`\`

### Teste falhou: "LocalStack pod not found"

Verifique se o LocalStack está instalado:

\`\`\`bash
kubectl get pods -n localstack
helm list -n localstack
\`\`\`

### Teste falhou: Resource não está READY

Verifique os logs do operator:

\`\`\`bash
kubectl logs -n iop-system deploy/infra-operator-controller-manager --tail=50
kubectl describe <resource-type> <resource-name>
\`\`\`

### Lambda ou RDS falham

Isso é esperado no LocalStack Free. Lambda pode ter issues intermitentes e RDS não está disponível.

## Cleanup

Os testes fazem cleanup automático ao terminar (via trap EXIT). Para fazer cleanup manual:

\`\`\`bash
kubectl delete vpc test-vpc
kubectl delete subnet test-subnet
kubectl delete s3bucket test-bucket-simple
kubectl delete dynamodbtable test-users-table
kubectl delete sqsqueue test-queue
kubectl delete snstopic test-topic
kubectl delete lambdafunction test-lambda
kubectl delete rdsinstance test-rds
\`\`\`

## Extensão dos Testes

Para adicionar novos testes, adicione uma função \`test_<resource>()\` seguindo o padrão:

\`\`\`bash
test_new_resource() {
    log_info "Test: New Resource description"

    if ! check_resource_exists <resource-type> <resource-name>; then
        log_error "<Resource> not found"
        return 1
    fi

    if wait_for_ready <resource-type> <resource-name>; then
        # Verificações adicionais
        log_success "Test passed"
        return 0
    else
        return 1
    fi
}
\`\`\`

E chame a função em \`main()\`:

\`\`\`bash
test_new_resource || true
\`\`\`

## CI/CD Integration

Para integrar com CI/CD, use:

\`\`\`bash
#!/bin/bash
set -e

# Setup
./setup-localstack.sh
./deploy-operator.sh
./apply-samples.sh

# Run tests
./test/e2e/e2e_test.sh

# Exit code: 0 = success, 1 = failure
\`\`\`
