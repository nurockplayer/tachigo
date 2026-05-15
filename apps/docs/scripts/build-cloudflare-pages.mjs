import {execFileSync} from 'node:child_process'
import {cpSync, existsSync, mkdirSync, rmSync} from 'node:fs'
import {resolve} from 'node:path'

const docsRoot = process.cwd()
const buildDir = resolve(docsRoot, 'build')
const distDir = resolve(docsRoot, 'dist')
const tachigoDir = resolve(distDir, 'tachigo')

execFileSync('pnpm', ['run', 'build'], {
  cwd: docsRoot,
  stdio: 'inherit',
})

if (!existsSync(buildDir)) {
  throw new Error(`Expected Docusaurus build output at ${buildDir}`)
}

rmSync(distDir, {force: true, recursive: true})
mkdirSync(tachigoDir, {recursive: true})

cpSync(buildDir, distDir, {recursive: true})
cpSync(buildDir, tachigoDir, {recursive: true})

console.log('Prepared Cloudflare Pages output in dist/ and dist/tachigo/.')
