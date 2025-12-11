import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Infra Operator',
  tagline: 'Kubernetes Operator for AWS Infrastructure Management',
  favicon: 'img/favicon.svg',

  future: {
    v4: true,
  },

  url: 'https://infra-operator.runner.codes',
  baseUrl: '/',

  organizationName: 'andrebassi',
  projectName: 'infra-operator',

  onBrokenLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'pt-BR'],
    localeConfigs: {
      en: {
        htmlLang: 'en-US',
        label: 'English',
      },
      'pt-BR': {
        htmlLang: 'pt-BR',
        label: 'Português (Brasil)',
      },
    },
  },

  markdown: {
    format: 'md',
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  presets: [
    [
      'classic',
      {
        docs: {
          routeBasePath: '/',
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/andrebassi/infra-operator/tree/main/docs/docusaurus/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/social-card.png',
    colorMode: {
      defaultMode: 'dark',
      disableSwitch: false,
      respectPrefersColorScheme: false,
    },
    navbar: {
      logo: {
        alt: 'Infra Operator',
        src: 'img/light.svg',
        srcDark: 'img/dark.svg',
        href: '/',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          to: '/api-reference/overview',
          label: 'API Reference',
          position: 'left',
        },
        {
          to: '/services/networking/vpc',
          label: 'AWS Services',
          position: 'left',
        },
        {
          type: 'localeDropdown',
          position: 'right',
        },
        {
          href: 'https://github.com/andrebassi/infra-operator',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      copyright: `Copyright ${new Date().getFullYear()} Infra Operator.<br/>Developed and maintained by <a href="https://andrebassi.com.br" target="_blank" rel="noopener noreferrer">Andre Bassi</a>`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'json', 'go', 'yaml', 'typescript', 'hcl'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
