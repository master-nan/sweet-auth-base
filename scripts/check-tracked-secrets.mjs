#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const SECRET_ENV_NAMES = [
  'APP_DBS_PRIMARY_PASSWORD',
  'APP_REDIS_PASSWORD',
  'APP_SESSION_SECRET',
  'APP_CONF_SALT',
  'APP_BOOTSTRAP_ADMIN_PASSWORD',
  'APP_BOOTSTRAP_APPLICATION_SECRET',
  'POSTGRES_PASSWORD',
]

const productionDeploymentPath = (filePath) => (
  filePath === 'docker-compose.external.yml'
  || filePath.startsWith('.github/workflows/')
  || /(?:^|\/)(?:config[-_.]?pro|production)(?:[./_-]|$)/i.test(filePath)
)

const secretEnvAlternation = SECRET_ENV_NAMES.join('|')

export const SECRET_SCAN_RULES = [
  {
    id: 'private-key',
    description: 'private key material',
    pattern: /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/g,
  },
  {
    id: 'github-token',
    description: 'GitHub access token',
    pattern: /\b(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{20,})\b/g,
  },
  {
    id: 'aws-access-key',
    description: 'AWS access key identifier',
    pattern: /\b(?:AKIA|ASIA)[A-Z0-9]{16}\b/g,
  },
  {
    id: 'slack-token',
    description: 'Slack token',
    pattern: /\bxox[baprs]-[A-Za-z0-9-]{20,}\b/g,
  },
  {
    id: 'credential-uri',
    description: 'URI containing inline credentials',
    pattern: /\b(?:postgres(?:ql)?|redis(?:s)?|mysql|mongodb(?:\+srv)?):\/\/[^\s/:<>]+:[^\s@<>]+@/gi,
    matchFilter: (match) => {
      const authority = match[0].slice(match[0].indexOf('://') + 3, -1)
      const password = authority.slice(authority.indexOf(':') + 1)
      return !/^(?:\*+|x+|redacted|masked)$/i.test(password)
    },
  },
  {
    id: 'production-secret-default',
    description: 'fixed secret fallback in production deployment config',
    pathFilter: productionDeploymentPath,
    pattern: new RegExp(`\\$\\{(?:${secretEnvAlternation}):-[^}]+\\}`, 'g'),
  },
  {
    id: 'production-secret-literal',
    description: 'literal secret in production deployment config',
    pathFilter: productionDeploymentPath,
    pattern: new RegExp(`^\\s*(?:${secretEnvAlternation})\\s*[:=]\\s*(?!\\$\\{|<|replace[_-]?with|change[_-]?me)(?:["']?)[^\\s#"']{8,}`, 'gmi'),
  },
]

function lineNumberAt(content, index) {
  let line = 1
  for (let cursor = 0; cursor < index; cursor += 1) {
    if (content.charCodeAt(cursor) === 10) line += 1
  }
  return line
}

function hasInlineAllowance(content, line, ruleId) {
  const lines = content.split(/\r?\n/)
  return [lines[line - 2], lines[line - 1]].some((value) => (
    value?.includes(`secret-scan: allow ${ruleId}`)
  ))
}

export function scanTrackedEntries(entries, rules = SECRET_SCAN_RULES) {
  const findings = []
  for (const entry of entries) {
    if (!entry || typeof entry.path !== 'string' || typeof entry.content !== 'string') continue
    if (entry.content.includes('\0')) continue

    for (const rule of rules) {
      if (rule.pathFilter && !rule.pathFilter(entry.path)) continue
      const pattern = new RegExp(rule.pattern.source, rule.pattern.flags)
      for (const match of entry.content.matchAll(pattern)) {
        if (rule.matchFilter && !rule.matchFilter(match)) continue
        const line = lineNumberAt(entry.content, match.index)
        if (hasInlineAllowance(entry.content, line, rule.id)) continue
        findings.push({
          path: entry.path,
          line,
          rule: rule.id,
          description: rule.description,
        })
      }
    }
  }
  return findings
}

export function readTrackedEntries({ cwd = process.cwd() } = {}) {
  const output = execFileSync('git', ['ls-files', '-z'], {
    cwd,
    encoding: 'utf8',
    maxBuffer: 32 * 1024 * 1024,
  })
  return output
    .split('\0')
    .filter(Boolean)
    .flatMap((filePath) => {
      const absolutePath = path.join(cwd, filePath)
      if (!fs.existsSync(absolutePath) || !fs.lstatSync(absolutePath).isFile()) return []
      return [{ path: filePath, content: fs.readFileSync(absolutePath, 'utf8') }]
    })
}

export function runTrackedSecretScan(options = {}) {
  return scanTrackedEntries(readTrackedEntries(options))
}

function main() {
  let findings
  try {
    findings = runTrackedSecretScan()
  } catch (error) {
    console.error(`Tracked secret scan could not run: ${error.message}`)
    process.exitCode = 1
    return
  }

  if (findings.length === 0) {
    console.log('Tracked secret scan passed.')
    return
  }

  console.error(`Tracked secret scan found ${findings.length} potential secret(s):`)
  for (const finding of findings) {
    console.error(`- ${finding.path}:${finding.line} [${finding.rule}] ${finding.description}`)
  }
  console.error('Remove the credential or add a narrowly scoped "secret-scan: allow <rule>" comment with justification.')
  process.exitCode = 1
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  main()
}
