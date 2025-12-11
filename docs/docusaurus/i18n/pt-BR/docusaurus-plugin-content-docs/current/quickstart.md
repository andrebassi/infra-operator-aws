# Início Rápido - Infra Operator

Comece a usar o infra-operator em menos de 5 minutos!

## Pré-requisitos

- Go 1.21+
- kubectl
- Docker ou OrbStack
- Task (go-task)

Instale as ferramentas necessárias:
```bash
brew install go kubectl go-task
# Instale o OrbStack em https://orbstack.dev/
```

## 1. Configurar Ambiente

**Comando:**

```bash
# Navegue até o projeto
cd /Users/andrebassi/works/.solutions/operators/infra-operator

# Execute a configuração completa (instala ferramentas, inicia LocalStack)
task setup
```

## 2. Iniciar Desenvolvimento

Escolha seu fluxo de trabalho:

### Opção A: Desenvolvimento Local (Teste Rápido)

Execute o operador localmente sem implantação no Kubernetes:

```bash
task dev
```

### Opção B: Executar Contra o Cluster (Recomendado)

Execute o operador localmente, gerenciando recursos no seu cluster:

```bash
# Terminal 1: Iniciar operador
task run:local

# Terminal 2: Aplicar recursos
kubectl apply -f test/e2e/fixtures/01-awsprovider.yaml
kubectl apply -f test/e2e/fixtures/02-s3bucket.yaml

# Monitorar recursos
kubectl get awsproviders,s3buckets -w
```

### Opção C: Implantação Completa no Cluster (Similar a Produção)

Implante o operador como um Pod no Kubernetes:

```bash
task dev:full
```

## 3. Verificar se Tudo Funciona

**Comando:**

```bash
# Verificar se AWSProvider está pronto
kubectl get awsproviders
# NAME         REGION      ACCOUNT         READY
# localstack   us-east-1   000000000000    true

# Verificar se S3Bucket está pronto
kubectl get s3buckets
# NAME               BUCKET-NAME                        REGION      READY
# e2e-test-bucket    e2e-test-bucket-infra-operator    us-east-1   true

# Verificar se o bucket existe no LocalStack
task localstack:aws -- s3 ls
# 2025-01-22 10:30:45 e2e-test-bucket-infra-operator

# Verificar status detalhado
kubectl describe s3bucket e2e-test-bucket
```

## 4. Visualizar Logs

**Comando:**

```bash
# Se usando dev:full (operador no cluster)
task k8s:logs

# Se usando run:local (operador no host)
# Logs são transmitidos no terminal, também salvos em /tmp/log.txt
tail -f /tmp/log.txt
```

## 5. Testar Ciclo de Vida do Recurso

**Comando:**

```bash
# Criar um novo bucket
cat <<EOF | kubectl apply -f -
apiVersion: aws-infra-operator.runner.codes/v1alpha1
kind: S3Bucket
metadata:
  name: my-test-bucket
  namespace: default
spec:
  providerRef:
    name: localstack
  bucketName: my-test-bucket-$(date +%s)
  versioning:
    enabled: true
  encryption:
    algorithm: AES256
  publicAccessBlock:
    blockPublicAcls: true
    ignorePublicAcls: true
    blockPublicPolicy: true
    restrictPublicBuckets: true
  tags:
    test: quickstart
  deletionPolicy: Delete
EOF

# Monitorar a criação
kubectl get s3bucket my-test-bucket -w

# Verificar se existe no LocalStack
task localstack:aws -- s3 ls | grep my-test-bucket

# Deletar o bucket
kubectl delete s3bucket my-test-bucket

# Verificar se foi removido
task localstack:aws -- s3 ls | grep my-test-bucket
# (não deve retornar nada)
```

## 6. Executar Testes

**Comando:**

```bash
# Testes unitários (rápidos, sem dependências externas)
task test:unit

# Testes de integração (usa LocalStack)
task test:integration

# Testes E2E (stack completa)
task test:e2e

# Todos os testes
task test:all
```

## 7. Limpar

**Comando:**

```bash
# Remover recursos de exemplo
task samples:delete

# Parar LocalStack (mantém dados)
task localstack:stop

# Remover tudo (operador, CRDs, LocalStack)
task clean:all
```

## Próximos Passos

- **Leia o [Guia de Desenvolvimento](/advanced/development)** para fluxos de trabalho detalhados
- **Leia o [Guia de Clean Architecture](/advanced/clean-architecture)** para entender a base de código
- **Consulte [Serviços AWS](/services/networking/vpc)** para todos os serviços suportados
- **Implemente mais controllers** usando S3 como template
- **Adicione webhooks** para validação e mutação
- **Implante em produção** usando IRSA para acesso seguro à AWS

## Problemas Comuns

### LocalStack não inicia

**Comando:**

```bash
# Verificar se Docker está rodando
docker ps

# Reiniciar LocalStack
task localstack:restart

# Ver logs
task localstack:logs
```

### Operador não reconcilia

**Comando:**

```bash
# Verificar se operador está rodando
kubectl get pods -n infra-operator-system

# Verificar logs
task k8s:logs

# Reiniciar operador
task k8s:restart
```

### S3Bucket preso em NotReady

**Comando:**

```bash
# Verificar se AWSProvider está pronto
kubectl get awsproviders

# Verificar detalhes do bucket
kubectl describe s3bucket <name>

# Verificar logs do operador para erros
task k8s:logs | grep -i error
```

## Referência Rápida

**Comando:**

```bash
# Desenvolvimento
task dev                  # Desenvolvimento local
task run:local            # Executar contra cluster
task dev:full             # Implantação completa
task dev:quick            # Rebuild rápido

# Testes
task test:unit            # Testes unitários
task test:integration     # Testes de integração
task test:e2e             # Testes E2E

# Kubernetes
task k8s:deploy           # Implantar operador
task k8s:logs             # Ver logs
task k8s:status           # Mostrar status

# LocalStack
task localstack:start     # Iniciar LocalStack
task localstack:aws -- s3 ls  # AWS CLI

# Limpeza
task clean:all            # Limpar tudo
```

## Obtendo Ajuda

**Comando:**

```bash
# Listar todas as tasks disponíveis
task --list

# Mostrar ajuda detalhada
task help

# Mostrar fonte da task
task --summary <task-name>
```

Para mais informações detalhadas, veja:
- [Guia de Desenvolvimento](/advanced/development)
- [Clean Architecture](/advanced/clean-architecture)
- [Serviços AWS](/services/networking/vpc)
- [Solução de Problemas](/advanced/troubleshooting)
