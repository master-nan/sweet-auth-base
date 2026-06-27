import type { Configure } from 'src/api/services/basic'

export type PasswordRule = (val: unknown) => true | string

function hasWhitespace(s: string): boolean {
  return /\s/.test(s)
}

function hasLower(s: string): boolean {
  return /[a-z]/.test(s)
}

function hasUpper(s: string): boolean {
  return /[A-Z]/.test(s)
}

function hasDigit(s: string): boolean {
  return /\d/.test(s)
}

function hasSpecial(s: string): boolean {
  // 与后端保持一致：只要是非字母非数字（且非空白）都视为特殊字符
  return /[^A-Za-z0-9\s]/.test(s)
}

export type PasswordPolicyConfig = Pick<Configure, 'password_length' | 'password_complexity'> &
  Partial<Pick<Configure, 'password_policy'>>

export type PasswordPolicyLevel = 'low' | 'medium' | 'strong' | 'high' | 'custom'

export interface PasswordPolicyPreset {
  value: PasswordPolicyLevel
  label: string
  shortLabel: string
  minLen: number
  complexity: number
  description: string
  regexText: string
}

export const passwordPolicyPresets: PasswordPolicyPreset[] = [
  {
    value: 'low',
    label: '低：至少6位',
    shortLabel: '低',
    minLen: 6,
    complexity: 1,
    description: '仅校验长度，不限制字符组合，适合内网临时或测试环境。',
    regexText: '^\\S{6,}$',
  },
  {
    value: 'medium',
    label: '中：至少8位，字母+数字',
    shortLabel: '中',
    minLen: 8,
    complexity: 2,
    description: '至少包含字母和数字，不允许空白字符，适合普通管理后台。',
    regexText: '^(?=.*[A-Za-z])(?=.*\\d)\\S{8,}$',
  },
  {
    value: 'high',
    label: '高：至少12位，三类字符',
    shortLabel: '高',
    minLen: 12,
    complexity: 3,
    description: '至少包含三类字符，且必须包含字母和数字，适合生产和高权限账号。',
    regexText: '^(?=.*[A-Za-z])(?=.*\\d)(?:(?=.*[a-z])(?=.*[A-Z])|(?=.*[a-z])(?=.*[^A-Za-z0-9\\s])|(?=.*[A-Z])(?=.*[^A-Za-z0-9\\s])|(?=.*\\d)(?=.*[^A-Za-z0-9\\s]))\\S{12,}$',
  },
  {
    value: 'custom',
    label: '自定义：手动设置长度和组合',
    shortLabel: '自定义',
    minLen: 6,
    complexity: 1,
    description: '使用下方最小长度和复杂度配置，适合特殊业务规则。',
    regexText: '按自定义长度和复杂度动态校验',
  },
]

export const passwordPolicyOptions = passwordPolicyPresets.map((preset) => ({
  label: preset.label,
  value: preset.value,
  description: preset.description,
}))

export function normalizePasswordPolicy(policy?: string): PasswordPolicyLevel {
  const normalized = String(policy ?? '').trim().toLowerCase()
  if (normalized === 'strong') return 'high'
  if (['low', 'medium', 'high', 'custom'].includes(normalized)) {
    return normalized as PasswordPolicyLevel
  }
  return 'medium'
}

export function getPasswordPolicyPreset(policy?: string): PasswordPolicyPreset {
  const normalized = normalizePasswordPolicy(policy)
  return passwordPolicyPresets.find((preset) => preset.value === normalized) ?? passwordPolicyPresets[1]!
}

export function effectivePasswordPolicy(cfg: PasswordPolicyConfig): {
  minLen: number
  complexity: number
} {
  const normalizedPolicy = normalizePasswordPolicy(cfg.password_policy)
  if (normalizedPolicy === 'custom') {
    return {
      minLen: Math.max(6, Number(cfg.password_length) || 0),
      complexity: Math.max(1, Number(cfg.password_complexity) || 0),
    }
  }
  const preset = getPasswordPolicyPreset(normalizedPolicy)
  return {
    minLen: preset.minLen,
    complexity: preset.complexity,
  }
}

export function passwordPolicyRegexText(cfg: PasswordPolicyConfig): string {
  const normalizedPolicy = normalizePasswordPolicy(cfg.password_policy)
  if (normalizedPolicy !== 'custom') {
    return getPasswordPolicyPreset(normalizedPolicy).regexText
  }
  const { minLen, complexity } = effectivePasswordPolicy(cfg)
  if (complexity <= 1) {
    return `^\\S{${minLen},}$`
  }
  if (complexity === 2) {
    return `^(?=.*[A-Za-z])(?=.*\\d)\\S{${minLen},}$`
  }
  return `^(?=.*[A-Za-z])(?=.*\\d)(?:(?=.*[a-z])(?=.*[A-Z])|(?=.*[a-z])(?=.*[^A-Za-z0-9\\s])|(?=.*[A-Z])(?=.*[^A-Za-z0-9\\s])|(?=.*\\d)(?=.*[^A-Za-z0-9\\s]))\\S{${minLen},}$`
}

export function passwordPolicyDescription(cfg: PasswordPolicyConfig): string {
  const { minLen, complexity } = effectivePasswordPolicy(cfg)

  if (complexity <= 1) {
    return `至少 ${minLen} 位`
  }

  if (complexity === 2) {
    return `至少 ${minLen} 位，且包含字母与数字`
  }

  // complexity >= 3
  return `至少 ${minLen} 位，且至少包含三类字符（大写/小写/数字/特殊字符），并必须包含字母与数字`
}

export function validatePasswordByConfigure(
  password: string,
  cfg: PasswordPolicyConfig
): true | string {
  const pwd = String(password ?? '').trim()
  if (!pwd) return '密码不能为空'
  if (hasWhitespace(pwd)) return '密码不能包含空白字符'

  const { minLen, complexity } = effectivePasswordPolicy(cfg)
  if (pwd.length < minLen) return `密码长度不足：至少 ${minLen} 位`

  if (complexity <= 1) return true

  const lower = hasLower(pwd)
  const upper = hasUpper(pwd)
  const digit = hasDigit(pwd)
  const special = hasSpecial(pwd)

  if (complexity >= 2) {
    if (!(digit && (lower || upper))) {
      return '密码复杂度不足：至少包含字母和数字'
    }
  }

  if (complexity >= 3) {
    const classCount = [lower, upper, digit, special].filter(Boolean).length
    if (classCount < 3) {
      return '密码复杂度不足：至少包含三类字符（大写/小写/数字/特殊字符）'
    }
  }

  return true
}

export function buildPasswordRules(
  cfg: PasswordPolicyConfig
): PasswordRule[] {
  return [
    (val) => {
      const password = typeof val === 'string' ? val : ''
      const result = validatePasswordByConfigure(password, cfg)
      return result === true ? true : result
    },
  ]
}
