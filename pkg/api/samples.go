package api

// Samples contém exemplos de requests para documentação Swagger
// Estes exemplos são baseados nos arquivos em ./samples/

// ===== VPC =====

// SampleVPC exemplo de criação de VPC
var SampleVPC = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "VPC",
    "metadata": {
      "name": "main-vpc",
      "namespace": "infra-operator"
    },
    "spec": {
      "cidrBlock": "10.0.0.0/16",
      "enableDnsSupport": true,
      "enableDnsHostnames": true
    }
  }]
}`

// ===== Subnet =====

// SampleSubnet exemplo de criação de Subnet
var SampleSubnet = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "Subnet",
    "metadata": {
      "name": "public-subnet-1a",
      "namespace": "infra-operator"
    },
    "spec": {
      "vpcId": "vpc-0abc123def456",
      "cidrBlock": "10.0.1.0/24",
      "availabilityZone": "us-east-1a",
      "mapPublicIpOnLaunch": true
    }
  }]
}`

// ===== SecurityGroup =====

// SampleSecurityGroup exemplo de criação de Security Group
var SampleSecurityGroup = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "SecurityGroup",
    "metadata": {
      "name": "web-sg",
      "namespace": "infra-operator"
    },
    "spec": {
      "vpcId": "vpc-0abc123def456",
      "description": "Security group para servidores web",
      "ingressRules": [
        {
          "protocol": "tcp",
          "fromPort": 80,
          "toPort": 80,
          "cidrIp": "0.0.0.0/0"
        },
        {
          "protocol": "tcp",
          "fromPort": 443,
          "toPort": 443,
          "cidrIp": "0.0.0.0/0"
        },
        {
          "protocol": "tcp",
          "fromPort": 22,
          "toPort": 22,
          "cidrIp": "10.0.0.0/8"
        }
      ]
    }
  }]
}`

// ===== EC2Instance =====

// SampleEC2Instance exemplo de criação de EC2
var SampleEC2Instance = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "EC2Instance",
    "metadata": {
      "name": "web-server-1",
      "namespace": "infra-operator"
    },
    "spec": {
      "instanceName": "web-server-1",
      "instanceType": "t3.micro",
      "imageId": "ami-0c55b159cbfafe1f0",
      "subnetId": "subnet-0abc123",
      "securityGroupIds": ["sg-0abc123"],
      "keyName": "my-keypair",
      "userData": "#!/bin/bash\nyum update -y\nyum install -y httpd\nsystemctl start httpd"
    }
  }]
}`

// ===== ComputeStack =====

// SampleComputeStackBasic exemplo básico de ComputeStack
var SampleComputeStackBasic = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "ComputeStack",
    "metadata": {
      "name": "dev-stack",
      "namespace": "infra-operator"
    },
    "spec": {
      "vpcCIDR": "10.100.0.0/16",
      "bastionInstance": {
        "enabled": true,
        "instanceType": "t3.micro"
      }
    }
  }]
}`

// SampleComputeStackWithCloudInit exemplo de ComputeStack com Cloud-Init
var SampleComputeStackWithCloudInit = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "ComputeStack",
    "metadata": {
      "name": "docker-stack",
      "namespace": "infra-operator"
    },
    "spec": {
      "vpcCIDR": "10.200.0.0/16",
      "bastionInstance": {
        "enabled": true,
        "instanceType": "t3.small",
        "userData": "#!/bin/bash\nyum update -y\nyum install -y docker\nsystemctl enable docker\nsystemctl start docker\nusermod -aG docker ec2-user"
      }
    }
  }]
}`

// ===== S3Bucket =====

// SampleS3Bucket exemplo de criação de S3 Bucket
var SampleS3Bucket = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "S3Bucket",
    "metadata": {
      "name": "my-app-bucket",
      "namespace": "infra-operator"
    },
    "spec": {
      "bucketName": "my-company-app-data-2024",
      "versioning": true,
      "encryption": {
        "enabled": true,
        "algorithm": "AES256"
      }
    }
  }]
}`

// ===== RDSInstance =====

// SampleRDSInstance exemplo de criação de RDS
var SampleRDSInstance = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "RDSInstance",
    "metadata": {
      "name": "app-database",
      "namespace": "infra-operator"
    },
    "spec": {
      "dbInstanceIdentifier": "app-postgres",
      "dbInstanceClass": "db.t3.micro",
      "engine": "postgres",
      "engineVersion": "15.4",
      "allocatedStorage": 20,
      "masterUsername": "admin",
      "masterPasswordSecretRef": {
        "name": "rds-credentials",
        "key": "password"
      },
      "vpcSecurityGroupIds": ["sg-0abc123"],
      "dbSubnetGroupName": "my-db-subnet-group"
    }
  }]
}`

// ===== DynamoDBTable =====

// SampleDynamoDBTable exemplo de criação de DynamoDB
var SampleDynamoDBTable = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "DynamoDBTable",
    "metadata": {
      "name": "users-table",
      "namespace": "infra-operator"
    },
    "spec": {
      "tableName": "Users",
      "attributeDefinitions": [
        {"attributeName": "userId", "attributeType": "S"},
        {"attributeName": "email", "attributeType": "S"}
      ],
      "keySchema": [
        {"attributeName": "userId", "keyType": "HASH"}
      ],
      "globalSecondaryIndexes": [
        {
          "indexName": "EmailIndex",
          "keySchema": [{"attributeName": "email", "keyType": "HASH"}],
          "projection": {"projectionType": "ALL"}
        }
      ],
      "billingMode": "PAY_PER_REQUEST"
    }
  }]
}`

// ===== SQSQueue =====

// SampleSQSQueue exemplo de criação de SQS
var SampleSQSQueue = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "SQSQueue",
    "metadata": {
      "name": "orders-queue",
      "namespace": "infra-operator"
    },
    "spec": {
      "queueName": "orders-processing",
      "visibilityTimeout": 30,
      "messageRetentionPeriod": 345600,
      "delaySeconds": 0,
      "fifoQueue": false
    }
  }]
}`

// ===== SNSTopic =====

// SampleSNSTopic exemplo de criação de SNS
var SampleSNSTopic = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "SNSTopic",
    "metadata": {
      "name": "notifications",
      "namespace": "infra-operator"
    },
    "spec": {
      "topicName": "app-notifications",
      "displayName": "App Notifications"
    }
  }]
}`

// ===== LambdaFunction =====

// SampleLambdaFunction exemplo de criação de Lambda
var SampleLambdaFunction = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "LambdaFunction",
    "metadata": {
      "name": "process-orders",
      "namespace": "infra-operator"
    },
    "spec": {
      "functionName": "process-orders",
      "runtime": "nodejs18.x",
      "handler": "index.handler",
      "role": "arn:aws:iam::123456789:role/lambda-execution-role",
      "codeS3Bucket": "my-lambda-code",
      "codeS3Key": "process-orders.zip",
      "memorySize": 256,
      "timeout": 30,
      "environment": {
        "QUEUE_URL": "https://sqs.us-east-1.amazonaws.com/123456789/orders"
      }
    }
  }]
}`

// ===== EKSCluster =====

// SampleEKSCluster exemplo de criação de EKS
var SampleEKSCluster = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "EKSCluster",
    "metadata": {
      "name": "production-cluster",
      "namespace": "infra-operator"
    },
    "spec": {
      "name": "production",
      "version": "1.28",
      "roleArn": "arn:aws:iam::123456789:role/eks-cluster-role",
      "vpcConfig": {
        "subnetIds": ["subnet-0abc123", "subnet-0def456"],
        "securityGroupIds": ["sg-0abc123"],
        "endpointPublicAccess": true,
        "endpointPrivateAccess": true
      }
    }
  }]
}`

// ===== IAMRole =====

// SampleIAMRole exemplo de criação de IAM Role
var SampleIAMRole = `{
  "resources": [{
    "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
    "kind": "IAMRole",
    "metadata": {
      "name": "lambda-role",
      "namespace": "infra-operator"
    },
    "spec": {
      "roleName": "lambda-execution-role",
      "assumeRolePolicyDocument": "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"lambda.amazonaws.com\"},\"Action\":\"sts:AssumeRole\"}]}",
      "managedPolicyArns": [
        "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
      ]
    }
  }]
}`

// ===== Multiple Resources =====

// SampleMultipleResources exemplo de múltiplos recursos
var SampleMultipleResources = `{
  "resources": [
    {
      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
      "kind": "VPC",
      "metadata": {"name": "app-vpc"},
      "spec": {"cidrBlock": "10.0.0.0/16"}
    },
    {
      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
      "kind": "Subnet",
      "metadata": {"name": "app-subnet"},
      "spec": {
        "vpcId": "vpc-xxx",
        "cidrBlock": "10.0.1.0/24",
        "availabilityZone": "us-east-1a"
      }
    },
    {
      "apiVersion": "aws-infra-operator.runner.codes/v1alpha1",
      "kind": "SecurityGroup",
      "metadata": {"name": "app-sg"},
      "spec": {
        "vpcId": "vpc-xxx",
        "description": "App Security Group"
      }
    }
  ]
}`

// GetAllSamples retorna todos os samples organizados por categoria
func GetAllSamples() map[string]map[string]string {
	return map[string]map[string]string{
		"Networking": {
			"VPC":           SampleVPC,
			"Subnet":        SampleSubnet,
			"SecurityGroup": SampleSecurityGroup,
		},
		"Compute": {
			"EC2Instance":              SampleEC2Instance,
			"ComputeStack (básico)":    SampleComputeStackBasic,
			"ComputeStack (Cloud-Init)": SampleComputeStackWithCloudInit,
			"EKSCluster":               SampleEKSCluster,
			"LambdaFunction":           SampleLambdaFunction,
		},
		"Database": {
			"RDSInstance":   SampleRDSInstance,
			"DynamoDBTable": SampleDynamoDBTable,
		},
		"Storage": {
			"S3Bucket": SampleS3Bucket,
		},
		"Messaging": {
			"SQSQueue": SampleSQSQueue,
			"SNSTopic": SampleSNSTopic,
		},
		"Security": {
			"IAMRole": SampleIAMRole,
		},
		"Multiple": {
			"VPC + Subnet + SecurityGroup": SampleMultipleResources,
		},
	}
}
