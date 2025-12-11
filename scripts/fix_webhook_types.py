#!/usr/bin/env python3

"""Script para corrigir nomes de tipos nos webhooks gerados"""

import os
import re
from pathlib import Path

API_DIR = "/Users/andrebassi/works/.solutions/operators/infra-operator/api/v1alpha1"

# Mapeamento de nomes incorretos para corretos
TYPE_MAPPINGS = {
    "Ualb": ("ALB", "alb"),
    "Uapigateway": ("APIGateway", "apigateway"),
    "Ucertificate": ("Certificate", "certificate"),
    "Ucloudfront": ("CloudFront", "cloudfront"),
    "Udynamodbtable": ("DynamoDBTable", "dynamodbtable"),
    "Uec2instance": ("EC2Instance", "ec2instance"),
    "Uecrrepository": ("ECRRepository", "ecrrepository"),
    "Uecscluster": ("ECSCluster", "ecscluster"),
    "Uekscluster": ("EKSCluster", "ekscluster"),
    "Uelasticachecluster": ("ElastiCacheCluster", "elasticachecluster"),
    "Uelasticip": ("ElasticIP", "elasticip"),
    "Uiamrole": ("IAMRole", "iamrole"),
    "Uinternetgateway": ("InternetGateway", "internetgateway"),
    "Ukmskey": ("KMSKey", "kmskey"),
    "Ulambdafunction": ("LambdaFunction", "lambdafunction"),
    "Unatgateway": ("NATGateway", "natgateway"),
    "Unlb": ("NLB", "nlb"),
    "Urdsinstance": ("RDSInstance", "rdsinstance"),
    "Uroute53hostedzone": ("Route53HostedZone", "route53hostedzone"),
    "Uroutetable": ("RouteTable", "routetable"),
    "Usecretsmanagersecret": ("SecretsManagerSecret", "secretsmanagersecret"),
    "Usecuritygroup": ("SecurityGroup", "securitygroup"),
    "Usnstopic": ("SNSTopic", "snstopic"),
    "Usqsqueue": ("SQSQueue", "sqsqueue"),
}

def fix_file(file_path, wrong, correct):
    """Fix type names in a file"""
    with open(file_path, 'r') as f:
        content = f.read()

    # Replace type references
    content = content.replace(f"*{wrong}", f"*{correct}")
    content = content.replace(f"&{wrong}", f"&{correct}")
    content = content.replace(f"validate{wrong}()", f"validate{correct}()")
    content = content.replace(f"func (r *{wrong})", f"func (r *{correct})")
    content = content.replace(f"var _ webhook.Validator = &{wrong}", f"var _ webhook.Validator = &{correct}")

    # Replace in test structs
    content = content.replace(f"var obj *{wrong}", f"var obj *{correct}")
    content = content.replace(f"obj = &{wrong}", f"obj = &{correct}")
    content = content.replace(f"{wrong}Spec", f"{correct}Spec")

    with open(file_path, 'w') as f:
        f.write(content)

def main():
    print("🔧 Corrigindo nomes de tipos nos webhooks...")
    print("")

    for wrong, (correct, lowercase) in TYPE_MAPPINGS.items():
        webhook_file = Path(API_DIR) / f"{lowercase}_webhook.go"
        test_file = Path(API_DIR) / f"{lowercase}_webhook_test.go"

        if webhook_file.exists():
            print(f"📝 Corrigindo {webhook_file.name}: {wrong} → {correct}")
            fix_file(webhook_file, wrong, correct)

        if test_file.exists():
            print(f"📝 Corrigindo {test_file.name}: {wrong} → {correct}")
            fix_file(test_file, wrong, correct)

    print("")
    print("✅ Tipos corrigidos com sucesso!")

if __name__ == "__main__":
    main()
