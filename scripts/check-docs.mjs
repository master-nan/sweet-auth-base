#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const docs = path.join(root, 'docs')
const expectedDocs = new Set([
  'README.md',
  'engineering/ExtensionDevelopmentGuide.md',
  'engineering/FrontendArchitectureGuide.md',
  'engineering/PlatformEngineeringGuide.md',
  'engineering/ProjectStructureGuide.md',
  'operations/PlatformOperationsGuide.md',
  'user-guide/DataPermissionUserGuide.md',
  'user-guide/FieldTypeGuide.md',
  'user-guide/LinkageConfig.md',
  'user-guide/LowCodeManual.md',
  'user-guide/OrganizationManagementUserGuide.md',
  'user-guide/PlatformAdministrationGuide.md',
  'user-guide/PlatformUserGuide.md',
])
const forbiddenReferences = [`docs/${'_'}construction`, 'docs/development']
const externalSchemes = ['http://', 'https://', 'mailto:', 'tel:', 'data:']

const walkFiles = (directory) =>
  fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const target = path.join(directory, entry.name)
    return entry.isDirectory() ? walkFiles(target) : [target]
  })

const withoutCode = (text) =>
  text.replace(/```[\s\S]*?```|~~~[\s\S]*?~~~/g, '').replace(/`[^`\n]+`/g, '')

const linkTargets = (text) => {
  const targets = []
  const patterns = [
    /!?\[[^\]]*\]\(([^)]+)\)/g,
    /^\s*\[[^\]]+\]:\s*(\S+)/gm,
    /\b(?:href|src)=["']([^"']+)["']/gi,
  ]
  for (const pattern of patterns) {
    for (const match of text.matchAll(pattern)) targets.push(match[1])
  }
  return targets
}

const localTarget = (source, rawTarget) => {
  let target = rawTarget.trim()
  if (!target || target.startsWith('#') || externalSchemes.some((scheme) => target.startsWith(scheme))) {
    return null
  }
  if (target.startsWith('<') && target.endsWith('>')) target = target.slice(1, -1)
  if (target.includes(' ')) target = target.split(' ', 1)[0]
  target = decodeURIComponent(target.split('#', 1)[0].split('?', 1)[0])
  return target ? path.resolve(path.dirname(source), target) : null
}

const errors = []
const actualDocs = new Set(
  walkFiles(docs)
    .filter((file) => path.basename(file) !== '.DS_Store')
    .map((file) => path.relative(docs, file).split(path.sep).join('/')),
)
const missing = [...expectedDocs].filter((file) => !actualDocs.has(file)).sort()
const unexpected = [...actualDocs].filter((file) => !expectedDocs.has(file)).sort()
if (missing.length) errors.push(`缺少必需文档：${missing.join(', ')}`)
if (unexpected.length) errors.push(`docs 中存在未登记文件：${unexpected.join(', ')}`)

const markdownFiles = [path.join(root, 'README.md'), ...walkFiles(docs).filter((file) => file.endsWith('.md'))]
for (const file of [...new Set(markdownFiles)].sort()) {
  if (!fs.existsSync(file)) {
    errors.push(`文档入口不存在：${path.relative(root, file)}`)
    continue
  }
  if (fs.statSync(file).size === 0) errors.push(`文档内容为空：${path.relative(root, file)}`)
  const text = withoutCode(fs.readFileSync(file, 'utf8'))
  for (const rawTarget of linkTargets(text)) {
    let target
    try {
      target = localTarget(file, rawTarget)
    } catch (error) {
      if (error instanceof URIError) {
        errors.push(`链接编码无效：${path.relative(root, file)} -> ${rawTarget}`)
        continue
      }
      throw error
    }
    if (target && !fs.existsSync(target)) {
      errors.push(`相对链接指向不存在：${path.relative(root, file)} -> ${rawTarget}`)
    }
  }
  for (const forbidden of forbiddenReferences) {
    if (text.includes(forbidden)) errors.push(`文档仍引用已删除路径：${path.relative(root, file)} -> ${forbidden}`)
  }
}

for (const relative of ['Makefile', 'AGENTS.md']) {
  const file = path.join(root, relative)
  if (!fs.existsSync(file)) continue
  const text = withoutCode(fs.readFileSync(file, 'utf8'))
  for (const forbidden of forbiddenReferences) {
    if (text.includes(forbidden)) errors.push(`文件仍引用已删除路径：${relative} -> ${forbidden}`)
  }
}

if (errors.length) {
  console.error('文档检查失败：')
  for (const error of errors) console.error(`- ${error}`)
  process.exit(1)
}

console.log(`文档检查通过：已检查 ${markdownFiles.length} 个 Markdown 文件。`)
