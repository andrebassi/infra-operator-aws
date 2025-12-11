import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: ['introduction', 'quickstart', 'installation', 'architecture'],
    },
    {
      type: 'category',
      label: 'Networking',
      items: [
        'services/networking/vpc',
        'services/networking/subnet',
        'services/networking/route-table',
        'services/networking/internet-gateway',
        'services/networking/nat-gateway',
        'services/networking/security-group',
        'services/networking/elastic-ip',
        'services/networking/alb',
        'services/networking/nlb',
      ],
    },
    {
      type: 'category',
      label: 'Compute',
      items: [
        'services/compute/ec2',
        'services/compute/eks',
        'services/compute/lambda',
        'services/compute/computestack',
      ],
    },
    {
      type: 'category',
      label: 'Storage',
      items: [
        'services/storage/s3',
      ],
    },
    {
      type: 'category',
      label: 'Database',
      items: [
        'services/database/rds',
        'services/database/dynamodb',
      ],
    },
    {
      type: 'category',
      label: 'Container',
      items: [
        'services/container/ecr',
        'services/container/ecs',
      ],
    },
    {
      type: 'category',
      label: 'Messaging',
      items: [
        'services/messaging/sqs',
        'services/messaging/sns',
      ],
    },
    {
      type: 'category',
      label: 'Security',
      items: [
        'services/security/iam',
        'services/security/secrets-manager',
        'services/security/kms',
        'services/security/acm',
      ],
    },
    {
      type: 'category',
      label: 'Caching',
      items: [
        'services/caching/elasticache',
      ],
    },
    {
      type: 'category',
      label: 'CDN & DNS',
      items: [
        'services/cdn/cloudfront',
        'services/dns/route53-hostedzone',
        'services/dns/route53-recordset',
      ],
    },
    {
      type: 'category',
      label: 'API Management',
      items: [
        'services/api/api-gateway',
      ],
    },
    {
      type: 'category',
      label: 'Features',
      items: [
        'features/cli',
        'features/api',
        'features/drift-detection',
        'features/prometheus-metrics',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api-reference/overview',
        'api-reference/awsprovider',
        'api-reference/crds',
      ],
    },
    {
      type: 'category',
      label: 'Advanced',
      items: [
        'advanced/clean-architecture',
        'advanced/development',
        'advanced/troubleshooting',
      ],
    },
  ],
};

export default sidebars;
