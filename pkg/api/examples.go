package api

// Exemplos de requests para Swagger
// Estes são usados nas anotações @x-examples dos handlers

// ===== Request Examples =====

// ExamplePlanVPC exemplo de plan para VPC
type ExamplePlanVPC struct {
	Resources []ExampleVPCResource `json:"resources" example:"[{...}]"`
}

// ExampleVPCResource exemplo de recurso VPC
type ExampleVPCResource struct {
	APIVersion string          `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string          `json:"kind" example:"VPC"`
	Metadata   ExampleMetadata `json:"metadata"`
	Spec       ExampleVPCSpec  `json:"spec"`
}

// ExampleMetadata exemplo de metadata
type ExampleMetadata struct {
	Name      string `json:"name" example:"main-vpc"`
	Namespace string `json:"namespace,omitempty" example:"infra-operator"`
}

// ExampleVPCSpec exemplo de spec de VPC
type ExampleVPCSpec struct {
	CidrBlock          string `json:"cidrBlock" example:"10.0.0.0/16"`
	EnableDnsSupport   bool   `json:"enableDnsSupport,omitempty" example:"true"`
	EnableDnsHostnames bool   `json:"enableDnsHostnames,omitempty" example:"true"`
}

// ExampleComputeStackResource exemplo de recurso ComputeStack
type ExampleComputeStackResource struct {
	APIVersion string                   `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string                   `json:"kind" example:"ComputeStack"`
	Metadata   ExampleMetadata          `json:"metadata"`
	Spec       ExampleComputeStackSpec  `json:"spec"`
}

// ExampleComputeStackSpec exemplo de spec de ComputeStack
type ExampleComputeStackSpec struct {
	VpcCIDR         string                 `json:"vpcCIDR" example:"10.100.0.0/16"`
	BastionInstance ExampleBastionInstance `json:"bastionInstance,omitempty"`
}

// ExampleBastionInstance exemplo de configuração de bastion
type ExampleBastionInstance struct {
	Enabled      bool   `json:"enabled" example:"true"`
	InstanceType string `json:"instanceType" example:"t3.micro"`
	UserData     string `json:"userData,omitempty" example:"#!/bin/bash\nyum install -y docker"`
}

// ExampleSubnetResource exemplo de recurso Subnet
type ExampleSubnetResource struct {
	APIVersion string             `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string             `json:"kind" example:"Subnet"`
	Metadata   ExampleMetadata    `json:"metadata"`
	Spec       ExampleSubnetSpec  `json:"spec"`
}

// ExampleSubnetSpec exemplo de spec de Subnet
type ExampleSubnetSpec struct {
	VpcId              string `json:"vpcId" example:"vpc-0abc123def456"`
	CidrBlock          string `json:"cidrBlock" example:"10.0.1.0/24"`
	AvailabilityZone   string `json:"availabilityZone" example:"us-east-1a"`
	MapPublicIpOnLaunch bool  `json:"mapPublicIpOnLaunch,omitempty" example:"true"`
}

// ExampleSecurityGroupResource exemplo de recurso SecurityGroup
type ExampleSecurityGroupResource struct {
	APIVersion string                    `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string                    `json:"kind" example:"SecurityGroup"`
	Metadata   ExampleMetadata           `json:"metadata"`
	Spec       ExampleSecurityGroupSpec  `json:"spec"`
}

// ExampleSecurityGroupSpec exemplo de spec de SecurityGroup
type ExampleSecurityGroupSpec struct {
	VpcId        string                 `json:"vpcId" example:"vpc-0abc123def456"`
	Description  string                 `json:"description" example:"Security group para web servers"`
	IngressRules []ExampleIngressRule   `json:"ingressRules,omitempty"`
}

// ExampleIngressRule exemplo de regra de ingress
type ExampleIngressRule struct {
	Protocol string `json:"protocol" example:"tcp"`
	FromPort int    `json:"fromPort" example:"443"`
	ToPort   int    `json:"toPort" example:"443"`
	CidrIp   string `json:"cidrIp" example:"0.0.0.0/0"`
}

// ExampleEC2Resource exemplo de recurso EC2Instance
type ExampleEC2Resource struct {
	APIVersion string           `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string           `json:"kind" example:"EC2Instance"`
	Metadata   ExampleMetadata  `json:"metadata"`
	Spec       ExampleEC2Spec   `json:"spec"`
}

// ExampleEC2Spec exemplo de spec de EC2Instance
type ExampleEC2Spec struct {
	InstanceName     string   `json:"instanceName" example:"web-server-1"`
	InstanceType     string   `json:"instanceType" example:"t3.micro"`
	ImageId          string   `json:"imageId" example:"ami-0c55b159cbfafe1f0"`
	SubnetId         string   `json:"subnetId" example:"subnet-0abc123"`
	SecurityGroupIds []string `json:"securityGroupIds" example:"sg-0abc123,sg-0def456"`
	KeyName          string   `json:"keyName,omitempty" example:"my-keypair"`
}

// ExampleS3Resource exemplo de recurso S3Bucket
type ExampleS3Resource struct {
	APIVersion string          `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string          `json:"kind" example:"S3Bucket"`
	Metadata   ExampleMetadata `json:"metadata"`
	Spec       ExampleS3Spec   `json:"spec"`
}

// ExampleS3Spec exemplo de spec de S3Bucket
type ExampleS3Spec struct {
	BucketName string `json:"bucketName" example:"my-company-data-bucket"`
	Versioning bool   `json:"versioning,omitempty" example:"true"`
}

// ExampleRDSResource exemplo de recurso RDSInstance
type ExampleRDSResource struct {
	APIVersion string           `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string           `json:"kind" example:"RDSInstance"`
	Metadata   ExampleMetadata  `json:"metadata"`
	Spec       ExampleRDSSpec   `json:"spec"`
}

// ExampleRDSSpec exemplo de spec de RDSInstance
type ExampleRDSSpec struct {
	DbInstanceIdentifier string `json:"dbInstanceIdentifier" example:"app-postgres"`
	DbInstanceClass      string `json:"dbInstanceClass" example:"db.t3.micro"`
	Engine               string `json:"engine" example:"postgres"`
	EngineVersion        string `json:"engineVersion" example:"15.4"`
	AllocatedStorage     int    `json:"allocatedStorage" example:"20"`
	MasterUsername       string `json:"masterUsername" example:"admin"`
}

// ExampleLambdaResource exemplo de recurso LambdaFunction
type ExampleLambdaResource struct {
	APIVersion string             `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string             `json:"kind" example:"LambdaFunction"`
	Metadata   ExampleMetadata    `json:"metadata"`
	Spec       ExampleLambdaSpec  `json:"spec"`
}

// ExampleLambdaSpec exemplo de spec de LambdaFunction
type ExampleLambdaSpec struct {
	FunctionName string `json:"functionName" example:"process-orders"`
	Runtime      string `json:"runtime" example:"nodejs18.x"`
	Handler      string `json:"handler" example:"index.handler"`
	Role         string `json:"role" example:"arn:aws:iam::123456789:role/lambda-role"`
	MemorySize   int    `json:"memorySize" example:"256"`
	Timeout      int    `json:"timeout" example:"30"`
}

// ExampleEKSResource exemplo de recurso EKSCluster
type ExampleEKSResource struct {
	APIVersion string           `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string           `json:"kind" example:"EKSCluster"`
	Metadata   ExampleMetadata  `json:"metadata"`
	Spec       ExampleEKSSpec   `json:"spec"`
}

// ExampleEKSSpec exemplo de spec de EKSCluster
type ExampleEKSSpec struct {
	Name    string `json:"name" example:"production"`
	Version string `json:"version" example:"1.28"`
	RoleArn string `json:"roleArn" example:"arn:aws:iam::123456789:role/eks-cluster-role"`
}

// ExampleDynamoDBResource exemplo de recurso DynamoDBTable
type ExampleDynamoDBResource struct {
	APIVersion string               `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string               `json:"kind" example:"DynamoDBTable"`
	Metadata   ExampleMetadata      `json:"metadata"`
	Spec       ExampleDynamoDBSpec  `json:"spec"`
}

// ExampleDynamoDBSpec exemplo de spec de DynamoDBTable
type ExampleDynamoDBSpec struct {
	TableName   string `json:"tableName" example:"Users"`
	BillingMode string `json:"billingMode" example:"PAY_PER_REQUEST"`
}

// ExampleSQSResource exemplo de recurso SQSQueue
type ExampleSQSResource struct {
	APIVersion string           `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string           `json:"kind" example:"SQSQueue"`
	Metadata   ExampleMetadata  `json:"metadata"`
	Spec       ExampleSQSSpec   `json:"spec"`
}

// ExampleSQSSpec exemplo de spec de SQSQueue
type ExampleSQSSpec struct {
	QueueName         string `json:"queueName" example:"orders-queue"`
	VisibilityTimeout int    `json:"visibilityTimeout" example:"30"`
	FifoQueue         bool   `json:"fifoQueue,omitempty" example:"false"`
}

// ExampleSNSResource exemplo de recurso SNSTopic
type ExampleSNSResource struct {
	APIVersion string           `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string           `json:"kind" example:"SNSTopic"`
	Metadata   ExampleMetadata  `json:"metadata"`
	Spec       ExampleSNSSpec   `json:"spec"`
}

// ExampleSNSSpec exemplo de spec de SNSTopic
type ExampleSNSSpec struct {
	TopicName   string `json:"topicName" example:"app-notifications"`
	DisplayName string `json:"displayName,omitempty" example:"App Notifications"`
}

// ExampleIAMRoleResource exemplo de recurso IAMRole
type ExampleIAMRoleResource struct {
	APIVersion string              `json:"apiVersion" example:"aws-infra-operator.runner.codes/v1alpha1"`
	Kind       string              `json:"kind" example:"IAMRole"`
	Metadata   ExampleMetadata     `json:"metadata"`
	Spec       ExampleIAMRoleSpec  `json:"spec"`
}

// ExampleIAMRoleSpec exemplo de spec de IAMRole
type ExampleIAMRoleSpec struct {
	RoleName                 string   `json:"roleName" example:"lambda-execution-role"`
	AssumeRolePolicyDocument string   `json:"assumeRolePolicyDocument" example:"{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"lambda.amazonaws.com\"},\"Action\":\"sts:AssumeRole\"}]}"`
	ManagedPolicyArns        []string `json:"managedPolicyArns,omitempty" example:"arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"`
}
