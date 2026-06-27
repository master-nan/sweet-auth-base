#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const DEFAULTS = {
  workspaceMinFreeMB: 2048,
  postgresMinFreeMB: 2048,
  buildCacheWarnMB: 10240,
}

export function parseSizeToMB(value) {
  const normalized = String(value || '').trim().replace(/,/g, '')
  const match = normalized.match(/^([0-9]+(?:\.[0-9]+)?)\s*([KMGT]?B)$/i)
  if (!match) return null
  const number = Number.parseFloat(match[1])
  const unit = match[2].toUpperCase()
  const factor = {
    B: 1 / 1024 / 1024,
    KB: 1 / 1024,
    MB: 1,
    GB: 1024,
    TB: 1024 * 1024,
  }[unit]
  return Math.round(number * factor)
}

export function parseDockerSystemDf(output) {
  return String(output || '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => JSON.parse(line))
}

export function parseDfPk(output) {
  const lines = String(output || '')
    .trim()
    .split(/\r?\n/)
    .filter(Boolean)
  if (lines.length < 2) {
    throw new Error('df output is incomplete')
  }
  const values = lines[lines.length - 1].trim().split(/\s+/)
  if (values.length < 6) {
    throw new Error(`df output row is invalid: ${lines[lines.length - 1]}`)
  }
  return {
    filesystem: values[0],
    totalMB: Math.floor(Number.parseInt(values[1], 10) / 1024),
    usedMB: Math.floor(Number.parseInt(values[2], 10) / 1024),
    availableMB: Math.floor(Number.parseInt(values[3], 10) / 1024),
    capacity: values[4],
    mount: values.slice(5).join(' '),
  }
}

export function parseComposePs(output) {
  return String(output || '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => JSON.parse(line))
}

export function validateServiceHealth(services, requiredServices) {
  const problems = []
  for (const serviceName of requiredServices) {
    const service = services.find((item) => item.Service === serviceName)
    if (!service) {
      problems.push(`compose service ${serviceName} is not present`)
      continue
    }
    if (service.State !== 'running') {
      problems.push(`compose service ${serviceName} is ${service.State || 'not running'}`)
      continue
    }
    if (service.Health && service.Health !== 'healthy') {
      problems.push(`compose service ${serviceName} health is ${service.Health}`)
    }
  }
  return problems
}

export function evaluateDiskCheck(name, disk, minFreeMB) {
  if (disk.availableMB < minFreeMB) {
    return `${name} has ${disk.availableMB}MB free; require at least ${minFreeMB}MB`
  }
  return ''
}

export function evaluateDockerCache(systemDfRows, warnMB) {
  const buildCache = systemDfRows.find((row) => row.Type === 'Build Cache')
  if (!buildCache) return ''
  const reclaimableText = String(buildCache.Reclaimable || '').split(/\s+/)[0]
  const reclaimableMB = parseSizeToMB(reclaimableText)
  if (reclaimableMB !== null && reclaimableMB >= warnMB) {
    return `Docker build cache reclaimable space is about ${reclaimableMB}MB; consider running docker builder prune before long release checks`
  }
  return ''
}

function envInteger(names, fallback) {
  const keys = Array.isArray(names) ? names : [names]
  const name = keys.find((key) => process.env[key] !== undefined && process.env[key] !== '') || keys[0]
  const value = process.env[name]
  if (value === undefined || value === '') return fallback
  const parsed = Number.parseInt(value, 10)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error(`${name} must be a positive integer`)
  }
  return parsed
}

function run(command, args, options = {}) {
  return execFileSync(command, args, {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
    ...options,
  })
}

function checkDockerReachable(problems) {
  try {
    run('docker', ['info', '--format', '{{json .ServerVersion}}'])
  } catch (error) {
    problems.push(`Docker is not reachable: ${String(error.stderr || error.message).trim()}`)
  }
}

function checkWorkspaceDisk(problems) {
  const minFreeMB = envInteger('SWEET_ADMIN_PREFLIGHT_MIN_WORKSPACE_MB', DEFAULTS.workspaceMinFreeMB)
  const disk = parseDfPk(run('df', ['-Pk', '.']))
  const problem = evaluateDiskCheck('workspace filesystem', disk, minFreeMB)
  if (problem) problems.push(problem)
}

function checkDockerCache(warnings) {
  const warnMB = envInteger('SWEET_ADMIN_PREFLIGHT_BUILD_CACHE_WARN_MB', DEFAULTS.buildCacheWarnMB)
  const warning = evaluateDockerCache(parseDockerSystemDf(run('docker', ['system', 'df', '--format', '{{json .}}'])), warnMB)
  if (warning) warnings.push(warning)
}

function checkComposeServices(problems) {
  const services = parseComposePs(run('docker', ['compose', 'ps', '--format', 'json']))
  problems.push(...validateServiceHealth(services, ['postgres', 'redis', 'backend', 'frontend']))
}

function checkPostgresDisk(problems) {
  const minFreeMB = envInteger('SWEET_ADMIN_PREFLIGHT_MIN_POSTGRES_MB', DEFAULTS.postgresMinFreeMB)
  const disk = parseDfPk(run('docker', ['compose', 'exec', '-T', 'postgres', 'sh', '-c', 'df -Pk /var/lib/postgresql/data']))
  const problem = evaluateDiskCheck('postgres /var/lib/postgresql/data filesystem', disk, minFreeMB)
  if (problem) problems.push(problem)
}

function printResult(label, problems, warnings) {
  if (problems.length > 0) {
    console.error(`${label} failed:`)
    for (const problem of problems) {
      console.error(`- ${problem}`)
    }
    process.exitCode = 1
    return
  }
  console.log(`${label} passed`)
  for (const warning of warnings) {
    console.warn(`Warning: ${warning}`)
  }
}

function main() {
  const mode = process.argv[2] || 'docker'
  const problems = []
  const warnings = []

  try {
    checkDockerReachable(problems)
    checkWorkspaceDisk(problems)
    if (problems.length === 0) {
      checkDockerCache(warnings)
    }
    if (mode === 'local') {
      checkComposeServices(problems)
      if (problems.length === 0) {
        checkPostgresDisk(problems)
      }
    } else if (mode !== 'docker') {
      problems.push('preflight-local mode must be docker or local')
    }
  } catch (error) {
    problems.push(String(error.stderr || error.message || error).trim())
  }

  printResult(mode === 'local' ? 'Local runtime preflight' : 'Docker preflight', problems, warnings)
}

const currentFile = fileURLToPath(import.meta.url)
if (process.argv[1] && path.resolve(process.argv[1]) === currentFile) {
  main()
}
