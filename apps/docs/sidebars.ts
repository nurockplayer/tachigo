import type {SidebarsConfig} from '@docusaurus/plugin-content-docs'

const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: 'doc',
      id: 'index',
      label: 'Dev Portal Home',
    },
    {
      type: 'category',
      label: 'Dev Portal',
      collapsed: false,
      items: [
        'dev-portal/start-here',
        'dev-portal/domain-maps',
        'dev-portal/daily-dev-guide',
        'dev-portal/flows',
        'dev-portal/source-index',
        'dev-portal/changelog',
        'dev-portal/graph-explorer',
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      collapsed: true,
      items: [
        'architecture',
        'auth-architecture',
        'sequence-diagram',
        'watch-to-points-design',
        'tokenomics',
        'backend-permissions',
      ],
    },
    {
      type: 'category',
      label: 'Policies',
      collapsed: true,
      items: [
        'auto-merge-policy',
        'dependabot-update-policy',
        'dependency-inventory-policy',
        'draft-pr-auto-ready',
        'pr-scope-policy',
        'uuid-v7',
      ],
    },
    {
      type: 'category',
      label: 'AI Workflow',
      collapsed: true,
      items: [
        'ai/README',
        'ai/autonomous-pr-gates',
        'ai/claude-codex-cheatsheet',
        'ai/claude-codex-workflow',
        'ai/codex-autonomous-workflow',
        'ai/github-actions-debugging',
        'ai/supply-chain-security',
        'ai/token-budget',
      ],
    },
    {
      type: 'category',
      label: 'Plans and Proposals',
      collapsed: true,
      items: [
        'atlas-migration-plan',
        'atlas-schema-reconciliation',
        'non-web3-launch-readiness',
        'openapi-codegen-flow',
        'superpowers/specs/2026-05-14-project-atlas-design',
      ],
    },
    {
      type: 'category',
      label: 'Reference Notes',
      collapsed: true,
      items: [
        'extension-ui-prompts',
        'feature-discussion',
        'tachimint-loyalty-claim-boundary',
        'history/2026-04-16-chrome-extension-terminology-audit',
        'history/2026-04-16-tachimint-chrome-sidepanel-migration',
        'history/2026-04-18-git-lfs-assets',
        'history/2026-04-30-monorepo-directory-refactor',
        'history/2026-05-01-dashboard-stack-evaluation',
      ],
    },
  ],
}

export default sidebars
