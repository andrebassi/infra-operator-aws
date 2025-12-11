#!/bin/bash

# Script para corrigir nomes de tipos nos webhooks gerados

API_DIR="/Users/andrebassi/works/.solutions/operators/infra-operator/api/v1alpha1"

# Mapeamento de nomes incorretos para corretos
declare -A TYPE_MAPPINGS=(
    ["Ualb"]="ALB"
    ["Uapigateway"]="APIGateway"
    ["Ucertificate"]="Certificate"
    ["Ucloudfront"]="CloudFront"
    ["Udynamodbtable"]="DynamoDBTable"
    ["Uec2instance"]="EC2Instance"
    ["Uecrrepository"]="ECRRepository"
    ["Uecscluster"]="ECSCluster"
    ["Uekscluster"]="EKSCluster"
    ["Uelasticachecluster"]="ElastiCacheCluster"
    ["Uelasticip"]="ElasticIP"
    ["Uiamrole"]="IAMRole"
    ["Uinternetgateway"]="InternetGateway"
    ["Ukmskey"]="KMSKey"
    ["Ulambdafunction"]="LambdaFunction"
    ["Unatgateway"]="NATGateway"
    ["Unlb"]="NLB"
    ["Urdsinstance"]="RDSInstance"
    ["Uroute53hostedzone"]="Route53HostedZone"
    ["Uroutetable"]="RouteTable"
    ["Usecretsmanagersecret"]="SecretsManagerSecret"
    ["Usecuritygroup"]="SecurityGroup"
    ["Usnstopic"]="SNSTopic"
    ["Usqsqueue"]="SQSQueue"
)

echo "🔧 Corrigindo nomes de tipos nos webhooks..."
echo ""

for wrong in "${!TYPE_MAPPINGS[@]}"; do
    correct="${TYPE_MAPPINGS[$wrong]}"
    lowercase=$(echo "$correct" | tr '[:upper:]' '[:lower:]' | sed 's/api/api/; s/iam/iam/; s/kms/kms/; s/ecr/ecr/; s/ecs/ecs/; s/eks/eks/; s/alb/alb/; s/nlb/nlb/; s/db/db/; s/ip/ip/; s/sqs/sqs/; s/sns/sns/')

    # Padrões de busca específicos baseados no tipo
    if [[ "$correct" == "ALB" ]]; then
        lowercase="alb"
    elif [[ "$correct" == "NLB" ]]; then
        lowercase="nlb"
    elif [[ "$correct" == "APIGateway" ]]; then
        lowercase="apigateway"
    elif [[ "$correct" == "IAMRole" ]]; then
        lowercase="iamrole"
    elif [[ "$correct" == "KMSKey" ]]; then
        lowercase="kmskey"
    elif [[ "$correct" == "ECRRepository" ]]; then
        lowercase="ecrrepository"
    elif [[ "$correct" == "ECSCluster" ]]; then
        lowercase="ecscluster"
    elif [[ "$correct" == "EKSCluster" ]]; then
        lowercase="ekscluster"
    elif [[ "$correct" == "EC2Instance" ]]; then
        lowercase="ec2instance"
    elif [[ "$correct" == "DynamoDBTable" ]]; then
        lowercase="dynamodbtable"
    elif [[ "$correct" == "ElasticIP" ]]; then
        lowercase="elasticip"
    elif [[ "$correct" == "ElastiCacheCluster" ]]; then
        lowercase="elasticachecluster"
    elif [[ "$correct" == "RDSInstance" ]]; then
        lowercase="rdsinstance"
    elif [[ "$correct" == "SNSTopic" ]]; then
        lowercase="snstopic"
    elif [[ "$correct" == "SQSQueue" ]]; then
        lowercase="sqsqueue"
    elif [[ "$correct" == "Certificate" ]]; then
        lowercase="certificate"
    elif [[ "$correct" == "CloudFront" ]]; then
        lowercase="cloudfront"
    elif [[ "$correct" == "InternetGateway" ]]; then
        lowercase="internetgateway"
    elif [[ "$correct" == "NATGateway" ]]; then
        lowercase="natgateway"
    elif [[ "$correct" == "Route53HostedZone" ]]; then
        lowercase="route53hostedzone"
    elif [[ "$correct" == "RouteTable" ]]; then
        lowercase="routetable"
    elif [[ "$correct" == "SecretsManagerSecret" ]]; then
        lowercase="secretsmanagersecret"
    elif [[ "$correct" == "SecurityGroup" ]]; then
        lowercase="securitygroup"
    elif [[ "$correct" == "LambdaFunction" ]]; then
        lowercase="lambdafunction"
    fi

    webhook_file="${API_DIR}/${lowercase}_webhook.go"
    test_file="${API_DIR}/${lowercase}_webhook_test.go"

    if [ -f "$webhook_file" ]; then
        echo "📝 Corrigindo $webhook_file: $wrong → $correct"
        sed -i.bak "s/$wrong/$correct/g" "$webhook_file"
        sed -i.bak "s/validate$wrong/validate$correct/g" "$webhook_file"
        rm "${webhook_file}.bak" 2>/dev/null || true
    fi

    if [ -f "$test_file" ]; then
        echo "📝 Corrigindo $test_file: $wrong → $correct"
        sed -i.bak "s/$wrong/$correct/g" "$test_file"
        rm "${test_file}.bak" 2>/dev/null || true
    fi
done

echo ""
echo "✅ Tipos corrigidos com sucesso!"
