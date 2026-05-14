import frontmatterValidatorPlugin from './plugins/frontmatter-validator/index.js'
import llmsTxtPlugin from './plugins/llms-txt/index.js'
import {themes as prismThemes} from 'prism-react-renderer'
import type {Config} from '@docusaurus/types'
import type * as Preset from '@docusaurus/preset-classic'

const config: Config = {
  title: 'tachigo Dev Portal',
  tagline: 'Project navigation for tachigo and tachiya',
  favicon: 'img/favicon.svg',

  url: 'https://nurockplayer.github.io',
  baseUrl: '/tachigo/',

  organizationName: 'nurockplayer',
  projectName: 'tachigo',
  trailingSlash: false,
  onBrokenLinks: 'throw',

  i18n: {
    defaultLocale: 'zh-Hant',
    locales: ['zh-Hant'],
  },

  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  themes: [
    '@docusaurus/theme-mermaid',
    [
      '@easyops-cn/docusaurus-search-local',
      {
        docsDir: '../../docs',
        docsRouteBasePath: '/',
        indexBlog: false,
        language: ['en', 'zh'],
        hashed: true,
        highlightSearchTermsOnTargetPage: true,
        searchResultLimits: 50,
      },
    ],
  ],
  plugins: [
    [
      frontmatterValidatorPlugin,
      {
        include: ['**/*.md'],
      },
    ],
    [
      llmsTxtPlugin,
      {
        include: ['**/*.md'],
      },
    ],
  ],

  presets: [
    [
      'classic',
      {
        docs: {
          path: '../../docs',
          routeBasePath: '/',
          sidebarPath: './sidebars.ts',
          exclude: ['README.md'],
          editUrl: ({docPath}) => {
            const repoDocPath = docPath
              .replace(/^(\.\.\/)+docs\//, 'docs/')
              .replace(/^(?!docs\/)/, 'docs/')
            return `https://github.com/nurockplayer/tachigo/edit/develop/${repoDocPath}`
          },
          showLastUpdateAuthor: true,
          showLastUpdateTime: true,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/tachigo-dev-portal-card.svg',
    navbar: {
      title: 'tachigo Dev Portal',
      logo: {
        alt: 'tachigo Dev Portal',
        src: 'img/favicon.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Guide',
        },
        {
          href: 'https://github.com/nurockplayer/tachigo',
          label: 'GitHub',
          position: 'right',
        },
        {
          href: 'https://github.com/nurockplayer/tachiya',
          label: 'tachiya',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Dev Portal',
          items: [
            {label: 'Start Here', to: '/dev-portal/start-here'},
            {label: 'Domain Maps', to: '/dev-portal/domain-maps'},
            {label: 'Daily Dev Guide', to: '/dev-portal/daily-dev-guide'},
          ],
        },
        {
          title: 'Repos',
          items: [
            {label: 'tachigo', href: 'https://github.com/nurockplayer/tachigo'},
            {label: 'tachiya', href: 'https://github.com/nurockplayer/tachiya'},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} tachigo contributors.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
}

export default config
