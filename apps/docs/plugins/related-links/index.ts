import type {LoadContext, Plugin} from '@docusaurus/types'

import {
  validateFrontmatterDocs,
  type FrontmatterValidatorOptions,
  type ValidationResult,
} from '../frontmatter-validator/index.ts'

export interface DocRelated {
  related_issues?: number[]
  implemented_in?: number[]
}

export interface RelatedLinksGlobalData {
  docs: Record<string, DocRelated>
}

export type RelatedLinksPluginOptions = Pick<
  FrontmatterValidatorOptions,
  'docsRoot' | 'include' | 'allowWarnings'
>

function copyNumbers(values: number[] | undefined): number[] | undefined {
  if (!values || values.length === 0) {
    return undefined
  }

  return [...values]
}

function hasRelatedLinks(related: DocRelated): boolean {
  return Boolean(related.related_issues?.length || related.implemented_in?.length)
}

export function collectRelatedLinks(validation: ValidationResult): RelatedLinksGlobalData {
  const docs: Record<string, DocRelated> = {}

  for (const file of validation.files) {
    const related: DocRelated = {}
    const relatedIssues = copyNumbers(file.frontmatter.related_issues)
    const implementedIn = copyNumbers(file.frontmatter.implemented_in)

    if (relatedIssues) {
      related.related_issues = relatedIssues
    }

    if (implementedIn) {
      related.implemented_in = implementedIn
    }

    if (hasRelatedLinks(related)) {
      docs[file.relativePath] = related
    }
  }

  return {docs}
}

export default function relatedLinksPlugin(
  context: LoadContext,
  options: RelatedLinksPluginOptions = {},
): Plugin<RelatedLinksGlobalData> {
  return {
    name: 'tachigo-related-links',

    async loadContent() {
      const validation = await validateFrontmatterDocs({
        siteDir: context.siteDir,
        docsRoot: options.docsRoot,
        include: options.include,
        allowWarnings: options.allowWarnings ?? true,
      })

      return collectRelatedLinks(validation)
    },

    async contentLoaded({content, actions}) {
      actions.setGlobalData(content)
    },
  }
}
