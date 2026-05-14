import fs from 'node:fs/promises'
import path from 'node:path'
import {fileURLToPath} from 'node:url'

import type {LoadContext, Plugin} from '@docusaurus/types'
import fg from 'fast-glob'
import matter from 'gray-matter'
import {z} from 'zod'

const dateStringSchema = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/, 'expected YYYY-MM-DD')

const frontmatterSchema = z
  .object({
    status: z.enum(['active', 'proposed', 'deprecated']).optional(),
    owner: z.string().min(1).optional(),
    last_reviewed: dateStringSchema.optional(),
    source_of_truth: z.boolean().optional(),
    code_areas: z.array(z.string()).optional(),
    related_repos: z.array(z.string()).optional(),
    related_issues: z.array(z.number().int().positive()).optional(),
    implemented_in: z.array(z.number().int().positive()).optional(),
    implemented_at: dateStringSchema.optional(),
    impl_status: z.enum(['shipped', 'in_progress', 'proposed']).optional(),
    reverse_index_mode: z.enum(['inferred', 'explicit_only', 'disabled']).optional(),
    reverse_index_scope: z.array(z.string()).optional(),
    excluded_from_llms: z.boolean().optional(),
    internal: z.boolean().optional(),
  })
  .superRefine((frontmatter, ctx) => {
    if (!frontmatter.reverse_index_scope) {
      return
    }

    const codeAreas = frontmatter.code_areas ?? []

    for (const [index, scope] of frontmatter.reverse_index_scope.entries()) {
      const isValid = codeAreas.some(
        (codeArea) => scope === codeArea || scope.startsWith(`${codeArea}/`),
      )

      if (!isValid) {
        ctx.addIssue({
          code: 'custom',
          path: ['reverse_index_scope', index],
          message: 'must equal a code_areas value or be nested beneath one',
        })
      }
    }
  })

export type FrontmatterData = z.infer<typeof frontmatterSchema>

export interface FrontmatterValidatorOptions {
  docsRoot?: string | URL
  include?: string[]
  allowWarnings?: boolean
}

export interface ValidatedDocFile {
  absolutePath: string
  relativePath: string
  frontmatter: FrontmatterData
}

export interface ValidationResult {
  docsRoot: string
  files: ValidatedDocFile[]
  warnings: string[]
}

function normalizeDateScalar(value: unknown): unknown {
  if (value instanceof Date && !Number.isNaN(value.getTime())) {
    return value.toISOString().slice(0, 10)
  }

  return value
}

function normalizeFrontmatter(input: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(
    Object.entries(input).map(([key, value]) => [key, normalizeDateScalar(value)]),
  )
}

function resolveDocsRoot(siteDir: string, docsRoot?: string | URL): string {
  if (docsRoot instanceof URL) {
    return path.resolve(fileURLToPath(docsRoot))
  }

  if (typeof docsRoot === 'string') {
    return path.isAbsolute(docsRoot) ? docsRoot : path.resolve(siteDir, docsRoot)
  }

  return path.resolve(siteDir, '../../docs')
}

function formatIssues(relativePath: string, issues: z.ZodIssue[]): string[] {
  return issues.map((issue) => {
    const field = issue.path.length > 0 ? issue.path.join('.') : '(frontmatter)'
    return `${relativePath}: ${field} ${issue.message}`
  })
}

function isLocalWarningEscapeEnabled(): boolean {
  return process.env.CI !== 'true' && process.env.DOCS_FRONTMATTER_VALIDATOR_ALLOW_WARNINGS === '1'
}

export async function validateFrontmatterDocs(
  options: FrontmatterValidatorOptions & {siteDir?: string} = {},
): Promise<ValidationResult> {
  const siteDir = options.siteDir ?? process.cwd()
  const docsRoot = resolveDocsRoot(siteDir, options.docsRoot)
  const include = options.include ?? ['**/*.md']
  const relativePaths = (
    await fg(include, {
      cwd: docsRoot,
      onlyFiles: true,
    })
  ).sort()

  if (relativePaths.length === 0) {
    throw new Error(`Frontmatter validation failed: no markdown files found under ${docsRoot}`)
  }

  const files: ValidatedDocFile[] = []
  const warnings: string[] = []

  for (const relativePath of relativePaths) {
    const absolutePath = path.join(docsRoot, relativePath)
    const source = await fs.readFile(absolutePath, 'utf8')
    const parsed = matter(source)
    const normalized = normalizeFrontmatter(parsed.data as Record<string, unknown>)
    const result = frontmatterSchema.safeParse(normalized)

    if (!result.success) {
      warnings.push(...formatIssues(relativePath, result.error.issues))
      continue
    }

    files.push({
      absolutePath,
      relativePath,
      frontmatter: result.data,
    })
  }

  if (warnings.length > 0 && !options.allowWarnings) {
    throw new Error(`Frontmatter validation failed:\n${warnings.join('\n')}`)
  }

  return {
    docsRoot,
    files,
    warnings,
  }
}

export default function frontmatterValidatorPlugin(
  context: LoadContext,
  options: FrontmatterValidatorOptions = {},
): Plugin<ValidationResult> {
  return {
    name: 'tachigo-frontmatter-validator',

    async loadContent() {
      const result = await validateFrontmatterDocs({
        siteDir: context.siteDir,
        docsRoot: options.docsRoot,
        include: options.include,
        allowWarnings: options.allowWarnings ?? isLocalWarningEscapeEnabled(),
      })

      if (result.warnings.length > 0) {
        console.warn(
          [
            'Frontmatter validator warnings were allowed by local escape hatch:',
            ...result.warnings,
          ].join('\n'),
        )
      }

      return result
    },

    async contentLoaded({content, actions}) {
      actions.setGlobalData(content)
    },
  }
}
